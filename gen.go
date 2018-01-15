package main

import (
	"bytes"
	"fmt"
	"hash/crc64"
	"io/ioutil"
	"log"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type KeyValueFiled struct {
	Key, Value *descriptor.FieldDescriptorProto
}

type Generator struct {
	OutputBuffer bytes.Buffer
	CppBuffer    bytes.Buffer
	dumpFileName string
	dumpCppName  string
	dumpDescName string
	macroName    string
	msgTypes     map[string]*descriptor.DescriptorProto
	HashValue    uint64
	//keyField, valueField *descriptor.FieldDescriptorProto
	hashEntryMessages map[string]KeyValueFiled
	packageName       string
	entryClassInfos   []string
}

func (g *Generator) Verify(file *descriptor.FileDescriptorProto) bool {
	//log.Printf("####%s", file.GetName())
	g.hashEntryMessages = make(map[string]KeyValueFiled)
	for _, msg := range file.MessageType {
		kv := KeyValueFiled{}
		for _, field := range msg.GetField() {
			if nil != field.GetOptions() {
				opstr := strings.TrimSpace(field.GetOptions().String())
				if opstr == "51234:1" {
					if nil != kv.Key {
						log.Fatalf("Duplicate filed with option: [(Key) = true]")
						return false
					}
					kv.Key = field
				} else if opstr == "51235:1" {
					if nil != kv.Value {
						log.Fatalf("Duplicate filed with option:  [(Value) = true]")
						return false
					}
					kv.Value = field
				}
			}
		}
		if nil != kv.Key && nil != kv.Value {
			g.hashEntryMessages[msg.GetName()] = kv
		} else if nil != kv.Key || nil != kv.Value {
			log.Fatalf("Missing filed with option: [(Key) = true] or [(Value) = true]")
			return false
		}
	}
	if len(g.hashEntryMessages) == 0 {
		//log.Printf("Maybe missing message with fileds with Options [(Key) = true] or [(Value) = true]")
		return false
	}

	return true
}

func (g *Generator) BuildTypeNameMap(file *descriptor.FileDescriptorProto) {
	if nil == g.msgTypes {
		g.msgTypes = make(map[string]*descriptor.DescriptorProto)
	}
	dottedPkg := "." + file.GetPackage()
	for _, msg := range file.MessageType {
		name := dottedPkg + "." + msg.GetName()
		g.msgTypes[name] = msg
		for _, nest := range msg.NestedType {
			g.msgTypes[name+"."+nest.GetName()] = nest
		}
	}
}

func (g *Generator) getDesc(name string) *descriptor.DescriptorProto {
	desc, exist := g.msgTypes[name]
	if exist {
		return desc
	}
	return nil
}

func (g *Generator) NestMarshal(msg *descriptor.DescriptorProto) []byte {
	buf := &bytes.Buffer{}
	data, _ := proto.Marshal(msg)
	buf.Write(data)
	for _, f := range msg.Field {
		fdesc := g.getDesc(f.GetTypeName())
		if nil != fdesc {
			data = g.NestMarshal(fdesc)
			buf.Write(data)
		}
	}
	return buf.Bytes()
}

func (g *Generator) DumpFile() {
	ioutil.WriteFile(g.dumpFileName, g.OutputBuffer.Bytes(), 0666)
	ioutil.WriteFile(g.dumpCppName, g.CppBuffer.Bytes(), 0666)
	ioutil.WriteFile(g.dumpDescName, []byte(strings.Join(g.entryClassInfos, "\n")), 0666)
}

func (g *Generator) TypeName(name string) string {
	if len(name) == 0 {
		return name
	}
	if name[0] == '.' {
		name = name[1:]
	}
	ss := strings.Split(name, ".")
	last := ss[len(ss)-1]
	return last
}

func (g *Generator) DumpHeader(pbfile string) {
	fname := pbfile
	if strings.Contains(fname, "/") {
		idx := strings.LastIndex(fname, "/")
		fname = fname[idx+1 : len(fname)]
	}
	g.dumpFileName = pbfile + ".hpp"
	g.dumpCppName = pbfile + ".cpp"
	g.dumpDescName = pbfile + ".desc"
	g.macroName = strings.ToUpper(fname+".hpp") + "_"
	g.macroName = strings.Replace(g.macroName, ".", "_", -1)
	g.macroName = strings.Replace(g.macroName, "/", "_", -1)
	fmt.Fprintf(&g.OutputBuffer, "// Generated by the plugin protoc-gen-mmadata of protocol buffer compiler.  DO NOT EDIT!\n")
	fmt.Fprintf(&g.OutputBuffer, "//  source: %s\n\n", pbfile)

	fmt.Fprintf(&g.OutputBuffer, "#ifndef %s\n", g.macroName)
	fmt.Fprintf(&g.OutputBuffer, "#define %s\n", g.macroName)
	fmt.Fprintf(&g.OutputBuffer, "#include <iosfwd>\n")
	fmt.Fprintf(&g.OutputBuffer, "#include \"kcfg.hpp\"\n")
	fmt.Fprintf(&g.OutputBuffer, "#include \"mmdata.hpp\"\n")
	fmt.Fprintf(&g.OutputBuffer, "#include \"mmdata_kcfg.hpp\"\n\n")

	fmt.Fprintf(&g.CppBuffer, "#include <iostream>\n")
	fmt.Fprintf(&g.CppBuffer, "#include \"%s\"\n\n", g.dumpFileName)
	fmt.Fprintf(&g.CppBuffer, "#include \"mmdata_util.hpp\"\n\n")

}

func (g *Generator) Finish() {
	fmt.Fprintf(&g.OutputBuffer, "#endif /* %s */\n", g.macroName)
}

func (g *Generator) DumpNamespaceBegin(name string) (string, []string) {
	ss := strings.Split(name, ".")
	var tabs []string
	tab := ""

	for _, ns := range ss {
		fmt.Fprintf(&g.OutputBuffer, "%snamespace %s\n%s{\n", tab, ns, tab)
		fmt.Fprintf(&g.CppBuffer, "%snamespace %s\n%s{\n", tab, ns, tab)
		tabs = append(tabs, tab)
		tab = "    " + tab
		if len(g.packageName) == 0 {
			g.packageName = ns
		} else {
			g.packageName = g.packageName + "." + ns
		}
	}
	return tab, tabs
}

func (g *Generator) DumpTemplates(tab string) {
	// fmt.Fprintf(&g.OutputBuffer, "%stemplate<typename T>\n", tab)
	// fmt.Fprintf(&g.OutputBuffer, "%sstd::ostream& operator<<(std::ostream& out, const typename boost::container::vector<T, mmdata::Allocator<T> >& v){\n", tab)
	// fmt.Fprintf(&g.OutputBuffer, "%s    return mmdata::operator<<(out, v);\n", tab)
	// fmt.Fprintf(&g.OutputBuffer, "%s}\n\n", tab)
}

func (g *Generator) DumpNamespaceEnd(tabs []string) {
	for i := len(tabs) - 1; i >= 0; i-- {
		fmt.Fprintf(&g.OutputBuffer, "%s}\n", tabs[i])
		fmt.Fprintf(&g.CppBuffer, "%s}\n", tabs[i])
	}
}

func (g *Generator) getBaseFieldType(field *descriptor.FieldDescriptorProto) string {
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return "double"
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return "float"
		// Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT64 if
		// negative values are likely.
	case descriptor.FieldDescriptorProto_TYPE_INT64:
		return "int64_t"
	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		return "uint64_tn"
		// Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT32 if
		// negative values are likely.
	case descriptor.FieldDescriptorProto_TYPE_INT32:
		return "int32_t"
	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		return "uint64_t"
	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		return "uint32_t"
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return "bool"
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return "mmdata::SHMString"
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return "mmdata::SHMString"
	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		return "uint32_t"
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return field.GetTypeName()
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		return "int32_t"
	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return "int64_t"
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		return "int32_t"
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		return "int64_t"
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		return g.TypeName(field.GetTypeName())
	default:
		log.Fatalf("Not supported type:%v", field.GetTypeName())
	}
	return ""
}

func (g *Generator) getFieldType(field *descriptor.FieldDescriptorProto) string {
	if field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		isMap := false
		buf := &bytes.Buffer{}
		if field.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
			desc := g.getDesc(field.GetTypeName())
			if nil != desc && desc.GetOptions().GetMapEntry() {
				keyField, valField := desc.Field[0], desc.Field[1]
				fmt.Fprintf(buf, "mmdata::SHMHashMap<%s, %s>::Type", g.getBaseFieldType(keyField), g.getBaseFieldType(valField))
				isMap = true
			}
		}
		if !isMap {
			fmt.Fprintf(buf, "mmdata::SHMVector<%s>::Type", g.getBaseFieldType(field))
		}
		return buf.String()
	}
	return g.getBaseFieldType(field)
}

func (g *Generator) withDefaultValue(field *descriptor.FieldDescriptorProto) (string, bool) {
	if field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return "", false
	}
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return "0.0", true
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return "0.0", true
	case descriptor.FieldDescriptorProto_TYPE_INT64:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_INT32:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return "false", true
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return "", false
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return "", false
	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		return "", false
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return "", false
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		return "0", true
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		return "0", true
	default:
		log.Fatalf("Not supported type:%v", field.GetTypeName())
	}
	return "", false
}

func (g *Generator) isComplextType(field *descriptor.FieldDescriptorProto) bool {
	if field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		return true
	}
	switch field.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return false
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return false
	case descriptor.FieldDescriptorProto_TYPE_INT64:
		return false
	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		return false
	case descriptor.FieldDescriptorProto_TYPE_INT32:
		return false
	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		return false
	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		return false
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return false
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return true
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return true
	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		return false
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return false
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		return false
	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return false
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		return false
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		return false
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		return true
	default:
		log.Fatalf("Not supported type:%v", field.GetTypeName())
	}
	return false
}

func (g *Generator) dumpFieldType(buf *bytes.Buffer, field *descriptor.FieldDescriptorProto) {
	fmt.Fprintf(buf, g.getFieldType(field))
}

func (g *Generator) DumpMessage(msg *descriptor.DescriptorProto, currentTAB string) error {

	buf := &g.OutputBuffer

	tableType := ""
	kv, haveKeyFiled := g.hashEntryMessages[msg.GetName()]
	if haveKeyFiled {

		if g.isComplextType(kv.Key) {
			fmt.Fprintf(buf, "%sinline std::size_t hash_value(const %s& v)\n", currentTAB, g.getFieldType(kv.Key))
			fmt.Fprintf(buf, "%s{\n", currentTAB)
			desc := g.getDesc(kv.Key.GetTypeName())
			funcTab := currentTAB + "    "
			fmt.Fprintf(buf, "%sstd::size_t hash = 0;\n", funcTab)
			for _, kf := range desc.Field {
				fmt.Fprintf(buf, "%shash ^= boost::hash_value<%s>(v.%s);\n", funcTab, g.getFieldType(kf), kf.GetName())
			}

			fmt.Fprintf(buf, "%sreturn hash;\n", funcTab)
			fmt.Fprintf(buf, "%s}\n", currentTAB)

			fmt.Fprintf(buf, "%sinline bool operator==(const %s& a, const %s& b)\n", currentTAB, g.getFieldType(kv.Key), g.getFieldType(kv.Key))
			fmt.Fprintf(buf, "%s{\n", currentTAB)
			for _, kf := range desc.Field {
				fmt.Fprintf(buf, "%sif(!(a.%s == b.%s)) return false;\n", funcTab, kf.GetName(), kf.GetName())
			}
			fmt.Fprintf(buf, "%sreturn true;\n", funcTab)
			fmt.Fprintf(buf, "%s}\n", currentTAB)

		}

		tableType = fmt.Sprintf("%sTable", msg.GetName())
		fmt.Fprintf(buf, "%sstruct %s;\n", currentTAB, tableType)
	}

	fmt.Fprintf(buf, "%sstruct %s\n", currentTAB, msg.GetName())
	fmt.Fprintf(buf, "%s{\n", currentTAB)
	fieldTab := currentTAB + "    "
	var fields string

	if haveKeyFiled {
		fmt.Fprintf(buf, "%stypedef %s key_type;\n", fieldTab, g.getFieldType(kv.Key))
		fmt.Fprintf(buf, "%stypedef %s value_type;\n", fieldTab, g.getFieldType(kv.Value))
		fmt.Fprintf(buf, "%stypedef %s table_type;\n", fieldTab, tableType)
	}
	for i, field := range msg.GetField() {
		fmt.Fprintf(buf, "%s", fieldTab)
		if field == kv.Key {
			fmt.Fprintf(buf, "key_type")
		} else if field == kv.Value {
			fmt.Fprintf(buf, "value_type")
		} else {
			g.dumpFieldType(buf, field)
		}

		fmt.Fprintf(buf, " %s;\n", field.GetName())
		if i != 0 {
			fields = fields + ","
		}
		fields = fields + field.GetName()
	}
	fmt.Fprintf(buf, "\n%sKCFG_DEFINE_FIELDS(%s)\n", fieldTab, fields)

	//constructor
	fmt.Fprintf(buf, "\n%s%s(const mmdata::CharAllocator& alloc):", fieldTab, msg.GetName())
	firstInitParam := true
	for _, field := range msg.GetField() {
		if g.isComplextType(field) {
			if !firstInitParam {
				fmt.Fprintf(buf, ",")
			}
			fmt.Fprintf(buf, "%s(alloc)", field.GetName())
			firstInitParam = false
		} else {
			defaultInitVal, exist := g.withDefaultValue(field)
			if exist {
				if !firstInitParam {
					fmt.Fprintf(buf, ",")
				}
				fmt.Fprintf(buf, "%s(%s)", field.GetName(), defaultInitVal)
				firstInitParam = false
			}
		}
	}
	fmt.Fprintf(buf, "\n%s{}\n", fieldTab)

	//GetKey/GetValue
	if haveKeyFiled {
		fmt.Fprintf(buf, "\n%sconst key_type& GetKey() const { return %s; }\n", fieldTab, kv.Key.GetName())
		fmt.Fprintf(buf, "%sconst value_type& GetValue() const { return %s; }\n", fieldTab, kv.Value.GetName())
	}

	fmt.Fprintf(buf, "%s};\n\n", currentTAB)

	fmt.Fprintf(buf, "%sinline std::ostream& operator<<(std::ostream& os, const %s& v)\n", currentTAB, msg.GetName())
	funcTab := currentTAB + "    "
	fmt.Fprintf(buf, "%s{\n", currentTAB)
	fmt.Fprintf(buf, "%sos<<\"[%s:\";\n", funcTab, msg.GetName())
	for i, field := range msg.GetField() {

		if i > 0 {
			fmt.Fprintf(buf, "%sos<<\",%s=\"<< v.%s;\n", funcTab, field.GetName(), field.GetName())
		} else {
			fmt.Fprintf(buf, "%sos<<\"%s=\"<< v.%s;\n", funcTab, field.GetName(), field.GetName())
		}
	}
	fmt.Fprintf(buf, "%sos<<\"]\";\n", funcTab)
	fmt.Fprintf(buf, "%sreturn os;\n", funcTab)
	fmt.Fprintf(buf, "%s}\n\n", currentTAB)

	if haveKeyFiled {
		currentClass := fmt.Sprintf("%sTable", msg.GetName())
		parentClassType := currentClass + "Parent"
		parentClass := fmt.Sprintf("mmdata::SHMHashMap<%s, %s>::Type", g.getFieldType(kv.Key), g.getFieldType(kv.Value))
		fmt.Fprintf(buf, "%stypedef %s %s;\n", currentTAB, parentClass, parentClassType)
		fmt.Fprintf(buf, "\n%sstruct %s:public %s\n", currentTAB, currentClass, parentClassType)
		fmt.Fprintf(buf, "%s{\n", currentTAB)
		funcTab := currentTAB + "    "
		fmt.Fprintf(buf, "%s%s(const mmdata::CharAllocator& alloc):%s(alloc)\n", funcTab, currentClass, parentClassType)
		fmt.Fprintf(buf, "%s{\n", funcTab)
		fmt.Fprintf(buf, "%s}\n\n", funcTab)
		hashData := g.NestMarshal(msg)
		crcTable := crc64.MakeTable(123456789)
		//g.HashValue = crc64.Checksum(hashData, crcTable)
		fmt.Fprintf(buf, "%sstatic uint64_t GetHash() { return %dUL;} \n", funcTab, crc64.Checksum(hashData, crcTable))
		fmt.Fprintf(buf, "%s};\n", currentTAB)
		//fmt.Fprintf(buf, "\n%stypedef mmdata::SHMHashMap<%s, %s>::Type %sTable;\n", currentTAB, g.getFieldType(keyField), g.getFieldType(valueField), msg.GetName())

		builderClass := fmt.Sprintf("%sHelper", currentClass)
		fmt.Fprintf(&g.CppBuffer, "%sstruct %s\n", currentTAB, builderClass)
		fmt.Fprintf(&g.CppBuffer, "%s{\n", currentTAB)
		fmt.Fprintf(&g.CppBuffer, "%sstatic int64_t Build(mmdata::DataImageBuildOptions& options, uint64_t& hash, std::string& err)\n", funcTab)
		fmt.Fprintf(&g.CppBuffer, "%s{\n", funcTab)
		funcBodyTab := funcTab + "    "
		fmt.Fprintf(&g.CppBuffer, "%shash = %s::GetHash();\n", funcBodyTab, currentClass)
		fmt.Fprintf(&g.CppBuffer, "%smmdata::DataImageBuilder builder;\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sint64_t ret = builder.Build<%s>(options);\n", funcBodyTab, msg.GetName())
		fmt.Fprintf(&g.CppBuffer, "%serr = builder.err;\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sreturn ret;\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%s}\n\n", funcTab)

		fmt.Fprintf(&g.CppBuffer, "%sstatic int TestMemory(const void* mem, const std::string& json_key)\n", funcTab)
		fmt.Fprintf(&g.CppBuffer, "%s{\n", funcTab)
		fmt.Fprintf(&g.CppBuffer, "%srapidjson::Document d;\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sd.Parse<0>(json_key.c_str());\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sif(d.HasParseError()){\n", funcBodyTab)
		funcBodyTab2 := funcBodyTab + "    "
		fmt.Fprintf(&g.CppBuffer, "%sstd::cout<<\"Invalid json key:\"<<json_key<<std::endl;\n", funcBodyTab2)
		fmt.Fprintf(&g.CppBuffer, "%sreturn -1;\n", funcBodyTab2)
		fmt.Fprintf(&g.CppBuffer, "%s}\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%stypedef %s::table_type RootTable;\n", funcBodyTab, msg.GetName())
		fmt.Fprintf(&g.CppBuffer, "%smmdata::MMData buf;\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sconst RootTable* root = buf.LoadRootReadObject<RootTable>(mem);\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sif (NULL == root) return -1;\n", funcBodyTab)
		if g.isComplextType(kv.Key) {
			fmt.Fprintf(&g.CppBuffer, "%smmdata::CharAllocator alloc;\n", funcBodyTab)
			fmt.Fprintf(&g.CppBuffer, "%s%s::key_type key(alloc);\n", funcBodyTab, msg.GetName())
		} else {
			fmt.Fprintf(&g.CppBuffer, "%s%s::key_type key;\n", funcBodyTab, msg.GetName())
		}

		fmt.Fprintf(&g.CppBuffer, "%skcfg::Parse(d, \"\", key);\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sRootTable::const_iterator found = root->find(key);\n", funcBodyTab)

		fmt.Fprintf(&g.CppBuffer, "%sif(found != root->end()){\n", funcBodyTab)

		fmt.Fprintf(&g.CppBuffer, "%sstd::cout << \"Found entry \"<< found->first << \"->\" << found->second << std::endl;\n", funcBodyTab2)
		fmt.Fprintf(&g.CppBuffer, "%sreturn 0;\n", funcBodyTab2)
		fmt.Fprintf(&g.CppBuffer, "%s}\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sstd::cout << \"NO Entry found for jsno_key:\"<< json_key << \"&key_obj:\"<<key<<std::endl;\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%sreturn -1;\n", funcBodyTab)
		fmt.Fprintf(&g.CppBuffer, "%s}\n\n", funcTab)

		fmt.Fprintf(&g.CppBuffer, "%s};\n\n", currentTAB)
		builderName := msg.GetName()
		if len(g.packageName) > 0 {
			builderName = g.packageName + "." + builderName
		}
		fmt.Fprintf(&g.CppBuffer, "%sstatic mmdata::HelperFuncRegister %s_instance(\"%s\", %s::Build,%s::TestMemory, %s::GetHash());\n", currentTAB, msg.GetName(), builderName, builderClass, builderClass, currentClass)

		//kv.Key.GetTypeName()
		entryInfo := currentClass
		if len(g.packageName) > 0 {
			entryInfo = g.packageName + "." + entryInfo
		}
		g.entryClassInfos = append(g.entryClassInfos, entryInfo)
	}

	return nil
}
