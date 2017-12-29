# protoc-gen-mmdata

A  protocol buffer3 compiler plugin to generater source file for [mmdata](https://github.com/yinqiwen/mmdata)  

## Dependency

- [golang](https://golang.org/)
- [protocol buffer3](https://github.com/google/protobuf)
- [mmdata](https://github.com/yinqiwen/mmdata)  Read/write data structure for shared memory
- [kcfg](https://github.com/yinqiwen/kcfg)  Headonly C++ json config mapping library.


## Usage
- go get -t -u github.com/yinqiwen/protoc-gen-mmdata
- protoc --mmdata_out=./ -I`<protoc-gen-mmdata_dir>` -I`<protobuf_include_dir>` mydata.proto

## Example
```proto
syntax = "proto3";
import "mmdata_base.proto";

package RECMD_SHM_DATA_V3;

message WhiteListItem
{
    int64 testid = 1;
    int64 ruleid = 2;
}
    
    
message WhiteListData 
{
    string imei = 1   [(Key) = true];
    repeated WhiteListItem items = 2 [(Value) = true];
}
```
