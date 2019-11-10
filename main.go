package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/k0kubun/pp"
	"github.com/mitchellh/go-mruby"
	"github.com/tzmfreedom/go-soapforce"
	"github.com/tzmfreedom/metaforce"
)

var username string
var password string
var endpoint = "login.salesforce.com"
var objects = map[string]*Object{}
var client *soapforce.Client
var currentObject *Object
var currentSobjects = map[string]struct{}{}

type Object struct {
	Name       string
	Properties map[string]*Property
}

type Property struct {
	Type  string
	Name  string
	Extra map[string]interface{}
}

type Diff struct {
	NewObjects    []*Object
	DeleteObjects []string
	NewColumns    map[string][]*Property
	UpdateColumns map[string][]*Property
	DeleteColumns map[string][]string
}

func main() {
	loadFile()
	var err error
	currentSobjects, err = getSalesforceSchema()
	if err != nil {
		panic(err)
	}
	diff, err := getDiff(currentSobjects, objects)
	if err != nil {
		panic(err)
	}
	//pp.Println(diff)
	apply(diff)
}

func getSalesforceSchema() (map[string]struct{}, error) {
	client = soapforce.NewClient()
	_, err := client.Login(username, password)
	if err != nil {
		return nil, err
	}
	r, err := client.DescribeGlobal()
	if err != nil {
		return nil, err
	}
	currentSobjects := map[string]struct{}{}
	for _, sobj := range r.Sobjects {
		if sobj.Custom {
			currentSobjects[sobj.Name] = struct{}{}
		}
	}
	return currentSobjects, nil
}

func apply(diff *Diff) error {
	metaClient := metaforce.NewClient()
	metaClient.SetDebug(true)
	err := metaClient.Login(username, password)
	if err != nil {
		panic(err)
	}
	createMetadataList := []metaforce.MetadataInterface{}
	for _, newObj := range diff.NewObjects {
		customFields := make([]*metaforce.CustomField, len(newObj.Properties))
		i := 0
		for _, prop := range newObj.Properties {
			customFields[i] = &metaforce.CustomField{
				FullName:    prop.Name,
				Label:       prop.Name,
				Type:        metaforce.FieldType(prop.Type),
				Description: "",
				Length:      255,
			}
			i++
		}
		createMetadataList = append(createMetadataList, &metaforce.CustomObject{
			FullName:         newObj.Name,
			Label:            newObj.Name,
			DeploymentStatus: metaforce.DeploymentStatusDeployed,
			Type:             "CustomObject",
			Description:      "",
			NameField: &metaforce.CustomField{
				Label:  "Name",
				Length: 80,
				Type:   metaforce.FieldTypeText,
			},
			SharingModel: metaforce.SharingModelReadWrite,
			Fields:       customFields,
		})
	}
	debug(createMetadataList)
	if len(createMetadataList) > 0 {
		r, err := metaClient.CreateMetadata(createMetadataList)
		if err != nil {
			panic(err)
		}
		debug(r)
	}

	customFields := make([]*metaforce.CustomField, len(diff.NewColumns))
	for objectName, columns := range diff.NewColumns {
		for i, column := range columns {
			customFields[i] = &metaforce.CustomField{
				FullName:    fmt.Sprintf("%s.%s", objectName, column.Name),
				Label:       column.Name,
				Type:        metaforce.FieldType(column.Type),
				Description: "",
			}
		}
		r, err := metaClient.CreateMetadata(createMetadataList)
		if err != nil {
			panic(err)
		}
		debug(r)
	}

	//r, err = metaClient.CreateMetadata(newFields)
	//if err != nil {
	//	panic(err)
	//}
	//debug(r)

	debug(diff)
	return nil
}

func getDiff(currentSobjects map[string]struct{}, settings map[string]*Object) (*Diff, error) {
	newObjects := []*Object{}
	deleteObjects := []string{}
	newColumns := map[string][]*Property{}
	deleteColumns := map[string][]string{}
	for name, setting := range settings {
		if _, ok := currentSobjects[name]; !ok {
			newObjects = append(newObjects, setting)
		} else {
			// columns diff
			dsr, err := client.DescribeSObject(name)
			if err != nil {
				return nil, err
			}
			currentProperties := map[string]*Property{}
			for _, f := range dsr.Fields {
				extra := map[string]interface{}{}
				if string(*f.Type_) == "string" {
					extra["length"] = 80
				}
				currentProperties[f.Name] = &Property{
					Name:  f.Name,
					Type:  string(*f.Type_),
					Extra: extra,
				}
			}
			tmpNewColumns := []*Property{}
			tmpDeleteColumns := []string{}
			for name, currentProperty := range currentProperties {
				if _, ok := setting.Properties[name]; !ok {
					tmpDeleteColumns = append(tmpDeleteColumns, name)
				} else {
					// update columns
					settingProperty := setting.Properties[name]
					if currentProperty.Type != settingProperty.Type {
						continue
					}
					switch currentProperty.Type {
					case "string":
						if currentProperty.Extra["length"] != settingProperty.Extra["length"] {

						}
					}
				}
			}
			for name, property := range setting.Properties {
				if _, ok := currentProperties[name]; !ok {
					tmpNewColumns = append(tmpNewColumns, property)
				}
			}
			newColumns[name] = tmpNewColumns
			deleteColumns[name] = tmpDeleteColumns
		}
	}
	for name, _ := range currentSobjects {
		if _, ok := settings[name]; !ok {
			deleteObjects = append(deleteObjects, name)
		}
	}
	return &Diff{
		NewObjects:    newObjects,
		DeleteObjects: deleteObjects,
		NewColumns:    newColumns,
		DeleteColumns: deleteColumns,
	}, nil
}

func loadFile() {
	mrb := mruby.NewMrb()
	defer mrb.Close()

	defineDSL(mrb)

	envModule := mrb.DefineClass("ENV", mrb.ObjectClass())
	envModule.DefineClassMethod("[]", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		args := m.GetArgs()
		key := args[0].String()
		return mrb.StringValue(os.Getenv(key)), nil
	}, mruby.ArgsReq(1))

	// Let's call it and inspect the result
	b, err := ioutil.ReadFile("./Somefile")
	_, err = mrb.LoadString(string(b))
	if err != nil {
		panic(err.Error())
	}
}

func defineDSL(mrb *mruby.Mrb) {
	kernel := mrb.KernelModule()
	kernel.DefineMethod("username", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		username = m.GetArgs()[0].String()
		return nil, nil
	}, mruby.ArgsReq(1))
	kernel.DefineMethod("password", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		password = m.GetArgs()[0].String()
		return nil, nil
	}, mruby.ArgsReq(1))
	kernel.DefineMethod("object", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		args := m.GetArgs()
		name := args[0].String()
		//properties := args[1].Hash()

		currentObject = &Object{name, map[string]*Property{}}
		objects[name] = currentObject
		mrb.Yield(args[1])
		return nil, nil
	}, mruby.ArgsReq(2))
	kernel.DefineMethod("text", createFieldProc(metaforce.FieldTypeText), mruby.ArgsReq(2))
	kernel.DefineMethod("number", createFieldProc(metaforce.FieldTypeNumber), mruby.ArgsReq(2))
	kernel.DefineMethod("date", createFieldProc(metaforce.FieldTypeDate), mruby.ArgsReq(2))
	kernel.DefineMethod("auto_number", createFieldProc(metaforce.FieldTypeAutoNumber), mruby.ArgsReq(2))
	kernel.DefineMethod("checkbox", createFieldProc(metaforce.FieldTypeCheckbox), mruby.ArgsReq(2))
	kernel.DefineMethod("currency", createFieldProc(metaforce.FieldTypeCurrency), mruby.ArgsReq(2))
	kernel.DefineMethod("date_time", createFieldProc(metaforce.FieldTypeDateTime), mruby.ArgsReq(2))
	kernel.DefineMethod("email", createFieldProc(metaforce.FieldTypeEmail), mruby.ArgsReq(2))
	kernel.DefineMethod("url", createFieldProc(metaforce.FieldTypeUrl), mruby.ArgsReq(2))
	kernel.DefineMethod("lookup", createFieldProc(metaforce.FieldTypeLookup), mruby.ArgsReq(2))
	kernel.DefineMethod("url", createFieldProc(metaforce.FieldTypeLongTextArea), mruby.ArgsReq(2))
	kernel.DefineMethod("long_text_area", createFieldProc(metaforce.FieldTypeHtml), mruby.ArgsReq(2))
	kernel.DefineMethod("percent", createFieldProc(metaforce.FieldTypePercent), mruby.ArgsReq(2))
	kernel.DefineMethod("picklist", createFieldProc(metaforce.FieldTypePicklist), mruby.ArgsReq(2))
	kernel.DefineMethod("multiselect_picklist", createFieldProc(metaforce.FieldTypeMultiselectPicklist), mruby.ArgsReq(2))
	kernel.DefineMethod("phone", createFieldProc(metaforce.FieldTypePhone), mruby.ArgsReq(2))
	kernel.DefineMethod("summary", createFieldProc(metaforce.FieldTypeSummary), mruby.ArgsReq(2))
}

func createFieldProc(fieldType metaforce.FieldType) func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
	return func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		if currentObject == nil {
			panic("No target object")
		}
		args := m.GetArgs()
		name := args[0].String()
		extra := getExtra(args)
		currentObject.Properties[name] = &Property{
			Type:  string(fieldType),
			Name:  name,
			Extra: extra,
		}
		return nil, nil
	}
}

func getExtra(args []*mruby.MrbValue) map[string]interface{} {
	extra := map[string]interface{}{}
	if len(args) > 1 {
		properties := args[1].Hash()
		k, _ := properties.Keys()
		keys := k.Array()
		for i := 0; i < keys.Len(); i++ {
			key, _ := keys.Get(i)
			value, _ := properties.Get(key)
			extra[key.String()] = value.String()
		}
	}
	return extra
}

func debug(args ...interface{}) {
	pp.Println(args)
}
