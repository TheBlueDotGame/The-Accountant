// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.25.1
// source: notary.proto

package protobufcompiled

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_notary_proto protoreflect.FileDescriptor

var file_notary_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x6e, 0x6f, 0x74, 0x61, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b,
	0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x1a, 0x16, 0x63, 0x6f, 0x6d,
	0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x32, 0xb7, 0x03, 0x0a, 0x09, 0x4e, 0x6f, 0x74, 0x61, 0x72, 0x79, 0x41, 0x50, 0x49, 0x12, 0x39,
	0x0a, 0x05, 0x41, 0x6c, 0x69, 0x76, 0x65, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a,
	0x16, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x2e, 0x41, 0x6c,
	0x69, 0x76, 0x65, 0x44, 0x61, 0x74, 0x61, 0x22, 0x00, 0x12, 0x3d, 0x0a, 0x07, 0x50, 0x72, 0x6f,
	0x70, 0x6f, 0x73, 0x65, 0x12, 0x18, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74,
	0x69, 0x73, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x1a, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x3d, 0x0a, 0x07, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x72, 0x6d, 0x12, 0x18, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69,
	0x73, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x1a, 0x16, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x3b, 0x0a, 0x06, 0x52, 0x65, 0x6a, 0x65, 0x63,
	0x74, 0x12, 0x17, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x2e,
	0x53, 0x69, 0x67, 0x6e, 0x65, 0x64, 0x48, 0x61, 0x73, 0x68, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x22, 0x00, 0x12, 0x3f, 0x0a, 0x07, 0x57, 0x61, 0x69, 0x74, 0x69, 0x6e, 0x67, 0x12,
	0x17, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x2e, 0x53, 0x69,
	0x67, 0x6e, 0x65, 0x64, 0x48, 0x61, 0x73, 0x68, 0x1a, 0x19, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75,
	0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x22, 0x00, 0x12, 0x3c, 0x0a, 0x05, 0x53, 0x61, 0x76, 0x65, 0x64, 0x12, 0x17,
	0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x2e, 0x53, 0x69, 0x67,
	0x6e, 0x65, 0x64, 0x48, 0x61, 0x73, 0x68, 0x1a, 0x18, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74,
	0x61, 0x6e, 0x74, 0x69, 0x73, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x22, 0x00, 0x12, 0x35, 0x0a, 0x04, 0x44, 0x61, 0x74, 0x61, 0x12, 0x14, 0x2e, 0x63, 0x6f,
	0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x2e, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x1a, 0x15, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x2e,
	0x44, 0x61, 0x74, 0x61, 0x42, 0x6c, 0x6f, 0x62, 0x22, 0x00, 0x42, 0x36, 0x5a, 0x34, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x61, 0x72, 0x74, 0x6f, 0x73, 0x73,
	0x68, 0x2f, 0x43, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x61, 0x6e, 0x74, 0x69, 0x73, 0x2f, 0x73, 0x72,
	0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x63, 0x6f, 0x6d, 0x70, 0x69, 0x6c,
	0x65, 0x64, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_notary_proto_goTypes = []interface{}{
	(*emptypb.Empty)(nil), // 0: google.protobuf.Empty
	(*Transaction)(nil),   // 1: computantis.Transaction
	(*SignedHash)(nil),    // 2: computantis.SignedHash
	(*Address)(nil),       // 3: computantis.Address
	(*AliveData)(nil),     // 4: computantis.AliveData
	(*Transactions)(nil),  // 5: computantis.Transactions
	(*DataBlob)(nil),      // 6: computantis.DataBlob
}
var file_notary_proto_depIdxs = []int32{
	0, // 0: computantis.NotaryAPI.Alive:input_type -> google.protobuf.Empty
	1, // 1: computantis.NotaryAPI.Propose:input_type -> computantis.Transaction
	1, // 2: computantis.NotaryAPI.Confirm:input_type -> computantis.Transaction
	2, // 3: computantis.NotaryAPI.Reject:input_type -> computantis.SignedHash
	2, // 4: computantis.NotaryAPI.Waiting:input_type -> computantis.SignedHash
	2, // 5: computantis.NotaryAPI.Saved:input_type -> computantis.SignedHash
	3, // 6: computantis.NotaryAPI.Data:input_type -> computantis.Address
	4, // 7: computantis.NotaryAPI.Alive:output_type -> computantis.AliveData
	0, // 8: computantis.NotaryAPI.Propose:output_type -> google.protobuf.Empty
	0, // 9: computantis.NotaryAPI.Confirm:output_type -> google.protobuf.Empty
	0, // 10: computantis.NotaryAPI.Reject:output_type -> google.protobuf.Empty
	5, // 11: computantis.NotaryAPI.Waiting:output_type -> computantis.Transactions
	1, // 12: computantis.NotaryAPI.Saved:output_type -> computantis.Transaction
	6, // 13: computantis.NotaryAPI.Data:output_type -> computantis.DataBlob
	7, // [7:14] is the sub-list for method output_type
	0, // [0:7] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_notary_proto_init() }
func file_notary_proto_init() {
	if File_notary_proto != nil {
		return
	}
	file_computantistypes_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_notary_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_notary_proto_goTypes,
		DependencyIndexes: file_notary_proto_depIdxs,
	}.Build()
	File_notary_proto = out.File
	file_notary_proto_rawDesc = nil
	file_notary_proto_goTypes = nil
	file_notary_proto_depIdxs = nil
}
