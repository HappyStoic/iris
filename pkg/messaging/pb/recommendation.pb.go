// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.19.3
// source: recommendation.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RecommendationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metadata *MetaData `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	TargetId string    `protobuf:"bytes,2,opt,name=targetId,proto3" json:"targetId,omitempty"`
}

func (x *RecommendationRequest) Reset() {
	*x = RecommendationRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_recommendation_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RecommendationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RecommendationRequest) ProtoMessage() {}

func (x *RecommendationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_recommendation_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RecommendationRequest.ProtoReflect.Descriptor instead.
func (*RecommendationRequest) Descriptor() ([]byte, []int) {
	return file_recommendation_proto_rawDescGZIP(), []int{0}
}

func (x *RecommendationRequest) GetMetadata() *MetaData {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *RecommendationRequest) GetTargetId() string {
	if x != nil {
		return x.TargetId
	}
	return ""
}

type RecommendationResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metadata       *MetaData       `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	RequestId      string          `protobuf:"bytes,2,opt,name=requestId,proto3" json:"requestId,omitempty"`
	Recommendation *Recommendation `protobuf:"bytes,3,opt,name=recommendation,proto3" json:"recommendation,omitempty"`
}

func (x *RecommendationResponse) Reset() {
	*x = RecommendationResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_recommendation_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RecommendationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RecommendationResponse) ProtoMessage() {}

func (x *RecommendationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_recommendation_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RecommendationResponse.ProtoReflect.Descriptor instead.
func (*RecommendationResponse) Descriptor() ([]byte, []int) {
	return file_recommendation_proto_rawDescGZIP(), []int{1}
}

func (x *RecommendationResponse) GetMetadata() *MetaData {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *RecommendationResponse) GetRequestId() string {
	if x != nil {
		return x.RequestId
	}
	return ""
}

func (x *RecommendationResponse) GetRecommendation() *Recommendation {
	if x != nil {
		return x.Recommendation
	}
	return nil
}

type Recommendation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeId string `protobuf:"bytes,1,opt,name=nodeId,proto3" json:"nodeId,omitempty"`
	Value  string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *Recommendation) Reset() {
	*x = Recommendation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_recommendation_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Recommendation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Recommendation) ProtoMessage() {}

func (x *Recommendation) ProtoReflect() protoreflect.Message {
	mi := &file_recommendation_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Recommendation.ProtoReflect.Descriptor instead.
func (*Recommendation) Descriptor() ([]byte, []int) {
	return file_recommendation_proto_rawDescGZIP(), []int{2}
}

func (x *Recommendation) GetNodeId() string {
	if x != nil {
		return x.NodeId
	}
	return ""
}

func (x *Recommendation) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

var File_recommendation_proto protoreflect.FileDescriptor

var file_recommendation_proto_rawDesc = []byte{
	0x0a, 0x14, 0x72, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x70, 0x62, 0x1a, 0x0a, 0x62, 0x61, 0x73, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x5d, 0x0a, 0x15, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d,
	0x65, 0x6e, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x28, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x0c, 0x2e, 0x70, 0x62, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x52,
	0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1a, 0x0a, 0x08, 0x74, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x74, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x49, 0x64, 0x22, 0x9c, 0x01, 0x0a, 0x16, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d,
	0x65, 0x6e, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x28, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x70, 0x62, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61,
	0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1c, 0x0a, 0x09, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x72,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x12, 0x3a, 0x0a, 0x0e, 0x72, 0x65, 0x63, 0x6f,
	0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x12, 0x2e, 0x70, 0x62, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0e, 0x72, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x22, 0x3e, 0x0a, 0x0e, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e,
	0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x42, 0x14, 0x5a, 0x12, 0x2e, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x69, 0x6e, 0x67, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_recommendation_proto_rawDescOnce sync.Once
	file_recommendation_proto_rawDescData = file_recommendation_proto_rawDesc
)

func file_recommendation_proto_rawDescGZIP() []byte {
	file_recommendation_proto_rawDescOnce.Do(func() {
		file_recommendation_proto_rawDescData = protoimpl.X.CompressGZIP(file_recommendation_proto_rawDescData)
	})
	return file_recommendation_proto_rawDescData
}

var file_recommendation_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_recommendation_proto_goTypes = []interface{}{
	(*RecommendationRequest)(nil),  // 0: pb.RecommendationRequest
	(*RecommendationResponse)(nil), // 1: pb.RecommendationResponse
	(*Recommendation)(nil),         // 2: pb.Recommendation
	(*MetaData)(nil),               // 3: pb.MetaData
}
var file_recommendation_proto_depIdxs = []int32{
	3, // 0: pb.RecommendationRequest.metadata:type_name -> pb.MetaData
	3, // 1: pb.RecommendationResponse.metadata:type_name -> pb.MetaData
	2, // 2: pb.RecommendationResponse.recommendation:type_name -> pb.Recommendation
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_recommendation_proto_init() }
func file_recommendation_proto_init() {
	if File_recommendation_proto != nil {
		return
	}
	file_base_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_recommendation_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RecommendationRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_recommendation_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RecommendationResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_recommendation_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Recommendation); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_recommendation_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_recommendation_proto_goTypes,
		DependencyIndexes: file_recommendation_proto_depIdxs,
		MessageInfos:      file_recommendation_proto_msgTypes,
	}.Build()
	File_recommendation_proto = out.File
	file_recommendation_proto_rawDesc = nil
	file_recommendation_proto_goTypes = nil
	file_recommendation_proto_depIdxs = nil
}