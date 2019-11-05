package main

import (
	"io/ioutil"
	"os"

	"github.com/k0kubun/pp"
	"github.com/mitchellh/go-mruby"
)

var username string
var password string
var endpoint = "login.salesforce.com"
var objects = []*Object{}

type Object struct {
	Name       string
	Properties []*Property
}

type Property struct {
	Type string
	Name string
}

type Diff struct {
	NewObjects    []*Object
	DeleteObjects []*Object
	NewColumns    map[string][]*Property
	UpdateColumns map[string][]*Property
	DeleteColumns map[string][]*Property
}

func main() {
	loadFile()
	schema, err := getSalesforceSchema()
	if err != nil {
		panic(err)
	}
	diff, err := getDiff(schema, objects)
	if err != nil {
		panic(err)
	}
	apply(diff)
}

func getSalesforceSchema() ([]*Object, error) {
	return nil, nil
}

func apply(diff *Diff) error {
	return nil
}

func getDiff(schema []*Object, objects []*Object) (*Diff, error) {
	return nil, nil
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
		currentObject = &Object{name, []*Property{}}
		objects = append(objects, currentObject)
		mrb.Yield(args[1])
		return nil, nil
	}, mruby.ArgsReq(2))
	kernel.DefineMethod("string", func(m *mruby.Mrb, self *mruby.MrbValue) (mruby.Value, mruby.Value) {
		if currentObject == nil {
			panic("No target object")
		}
		args := m.GetArgs()
		name := args[0].String()
		currentObject.Properties = append(currentObject.Properties, &Property{
			Type: "string",
			Name: name,
		})
		return nil, nil
	}, mruby.ArgsReq(2))
}

func debug(args ...interface{}) {
	pp.Println(args)
}
