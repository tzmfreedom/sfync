package main

import (
	"io/ioutil"
	"os"

	"github.com/k0kubun/pp"
	"github.com/mitchellh/go-mruby"
	"github.com/tzmfreedom/go-soapforce"
)

var username string
var password string
var endpoint = "login.salesforce.com"
var objects = map[string]*Object{}
var client *soapforce.Client
var currentSobjects = map[string]struct{}{}

type Object struct {
	Name       string
	Properties map[string]*Property
}

type Property struct {
	Type string
	Name string
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
	apply(diff)
}

func getSalesforceSchema() (map[string]struct{}, error) {
	client = soapforce.NewClient()
	_, err := client.Login(os.Getenv("SFDC_USERNAME"), os.Getenv("SFDC_PASSWORD"))
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
				currentProperties[f.Name] = &Property{
					Name: f.Name,
					Type: string(*f.Type_),
				}
			}
			tmpNewColumns := []*Property{}
			tmpDeleteColumns := []string{}
			for name, _ := range currentProperties {
				if _, ok := setting.Properties[name]; !ok {
					tmpDeleteColumns = append(tmpDeleteColumns, name)
				}
			}
			for name, property := range setting.Properties {
				if _, ok := currentProperties[name]; !ok {
					tmpNewColumns = append(tmpNewColumns, property)
				}
			}
			newColumns[name] = tmpNewColumns
			deleteColumns[name] = tmpDeleteColumns
			//objects[name] = &Object{
			//	Name:       name,
			//	Properties: properties,
			//}
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
	var currentObject *Object
	kernel := mrb.KernelModule()
	kernel.DefineMethod("config", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		args := m.GetArgs()
		mrb.Yield(args[0])
		return nil, nil
	}, mruby.ArgsReq(1))
	kernel.DefineMethod("username", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		username = m.GetArgs()[0].String()
		return nil, nil
	}, mruby.ArgsReq(1))
	kernel.DefineMethod("password", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		password = m.GetArgs()[0].String()
		return nil, nil
	}, mruby.ArgsReq(1))
	kernel.DefineMethod("password", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		endpoint = m.GetArgs()[0].String()
		return nil, nil
	}, mruby.ArgsReq(1))
	kernel.DefineMethod("object", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		args := m.GetArgs()
		name := args[0].String()
		//properties := args[1].Hash()

		//k, _ := properties.Keys()
		//keys := k.Array()
		//for i := 0; i < keys.Len(); i++ {
		//	key, _ := keys.Get(i)
		//	value, _ := properties.Get(key)
		//	fmt.Printf("%s => %s\n", key.String(), value)
		//}
		currentObject = &Object{name, map[string]*Property{}}
		objects[name] = currentObject
		mrb.Yield(args[1])
		return nil, nil
	}, mruby.ArgsReq(2))
	kernel.DefineMethod("string", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		if currentObject == nil {
			panic("No target object")
		}
		args := m.GetArgs()
		name := args[0].String()
		currentObject.Properties[name] = &Property{
			Type: "string",
			Name: name,
		}
		return nil, nil
	}, mruby.ArgsReq(2))
}

func debug(args ...interface{}) {
	pp.Println(args)
}
