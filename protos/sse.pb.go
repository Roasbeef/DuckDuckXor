// Code generated by protoc-gen-go.
// source: sse.proto
// DO NOT EDIT!

/*
Package sse_protos is a generated protocol buffer package.

It is generated from these files:
	sse.proto

It has these top-level messages:
	IndexData
	IndexAck
	CipherDoc
	CipherDocAck
	KeywordQuery
	BooleanSearchQuery
	XTokenRequest
	XTokenResponse
	Error
*/
package sse_protos

import proto "github.com/golang/protobuf/proto"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

type IndexData_MessageType int32

const (
	IndexData_T_SET IndexData_MessageType = 0
	IndexData_X_SET IndexData_MessageType = 1
)

var IndexData_MessageType_name = map[int32]string{
	0: "T_SET",
	1: "X_SET",
}
var IndexData_MessageType_value = map[string]int32{
	"T_SET": 0,
	"X_SET": 1,
}

func (x IndexData_MessageType) String() string {
	return proto.EnumName(IndexData_MessageType_name, int32(x))
}

type BooleanSearchQuery_SearchType int32

const (
	BooleanSearchQuery_NEGATION    BooleanSearchQuery_SearchType = 0
	BooleanSearchQuery_SNF         BooleanSearchQuery_SearchType = 1
	BooleanSearchQuery_ARBITRARY   BooleanSearchQuery_SearchType = 2
	BooleanSearchQuery_DISJUNCTION BooleanSearchQuery_SearchType = 3
)

var BooleanSearchQuery_SearchType_name = map[int32]string{
	0: "NEGATION",
	1: "SNF",
	2: "ARBITRARY",
	3: "DISJUNCTION",
}
var BooleanSearchQuery_SearchType_value = map[string]int32{
	"NEGATION":    0,
	"SNF":         1,
	"ARBITRARY":   2,
	"DISJUNCTION": 3,
}

func (x BooleanSearchQuery_SearchType) String() string {
	return proto.EnumName(BooleanSearchQuery_SearchType_name, int32(x))
}

type IndexData struct {
	Type IndexData_MessageType `protobuf:"varint,1,opt,name=type,enum=sse_protos.IndexData_MessageType" json:"type,omitempty"`
	TSet *IndexData_TSet       `protobuf:"bytes,2,opt,name=t_set" json:"t_set,omitempty"`
	XSet *IndexData_XSet       `protobuf:"bytes,3,opt,name=x_set" json:"x_set,omitempty"`
}

func (m *IndexData) Reset()         { *m = IndexData{} }
func (m *IndexData) String() string { return proto.CompactTextString(m) }
func (*IndexData) ProtoMessage()    {}

func (m *IndexData) GetTSet() *IndexData_TSet {
	if m != nil {
		return m.TSet
	}
	return nil
}

func (m *IndexData) GetXSet() *IndexData_XSet {
	if m != nil {
		return m.XSet
	}
	return nil
}

type IndexData_TSet struct {
	// (b || L) -> (B || s_i) XOR K
	// TODO(roasbeef): Need to decide on length split for these vals
	// Key is hex-encoded byte array
	TTuples map[string][]byte `protobuf:"bytes,1,rep,name=t_tuples" json:"t_tuples,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (m *IndexData_TSet) Reset()         { *m = IndexData_TSet{} }
func (m *IndexData_TSet) String() string { return proto.CompactTextString(m) }
func (*IndexData_TSet) ProtoMessage()    {}

func (m *IndexData_TSet) GetTTuples() map[string][]byte {
	if m != nil {
		return m.TTuples
	}
	return nil
}

// g^( F_p(K_x, w) * xind )
type IndexData_XSet struct {
	Xtag []byte `protobuf:"bytes,1,opt,name=xtag,proto3" json:"xtag,omitempty"`
}

func (m *IndexData_XSet) Reset()         { *m = IndexData_XSet{} }
func (m *IndexData_XSet) String() string { return proto.CompactTextString(m) }
func (*IndexData_XSet) ProtoMessage()    {}

type IndexAck struct {
	Ack bool `protobuf:"varint,1,opt,name=ack" json:"ack,omitempty"`
}

func (m *IndexAck) Reset()         { *m = IndexAck{} }
func (m *IndexAck) String() string { return proto.CompactTextString(m) }
func (*IndexAck) ProtoMessage()    {}

type CipherDoc struct {
	// CipherDoc ID.
	DocId        int32  `protobuf:"varint,1,opt,name=doc_id" json:"doc_id,omitempty"`
	EncryptedDoc []byte `protobuf:"bytes,2,opt,name=encrypted_doc,proto3" json:"encrypted_doc,omitempty"`
}

func (m *CipherDoc) Reset()         { *m = CipherDoc{} }
func (m *CipherDoc) String() string { return proto.CompactTextString(m) }
func (*CipherDoc) ProtoMessage()    {}

type CipherDocAck struct {
	Ack bool `protobuf:"varint,1,opt,name=ack" json:"ack,omitempty"`
}

func (m *CipherDocAck) Reset()         { *m = CipherDocAck{} }
func (m *CipherDocAck) String() string { return proto.CompactTextString(m) }
func (*CipherDocAck) ProtoMessage()    {}

// stag = F(K_t, w)
type KeywordQuery struct {
	Stag []byte `protobuf:"bytes,1,opt,name=stag,proto3" json:"stag,omitempty"`
}

func (m *KeywordQuery) Reset()         { *m = KeywordQuery{} }
func (m *KeywordQuery) String() string { return proto.CompactTextString(m) }
func (*KeywordQuery) ProtoMessage()    {}

type BooleanSearchQuery struct {
	Type   BooleanSearchQuery_SearchType            `protobuf:"varint,1,opt,name=type,enum=sse_protos.BooleanSearchQuery_SearchType" json:"type,omitempty"`
	NQuery *BooleanSearchQuery_NegatedConjunction   `protobuf:"bytes,2,opt,name=n_query" json:"n_query,omitempty"`
	SQuery *BooleanSearchQuery_SearchableNormalForm `protobuf:"bytes,3,opt,name=s_query" json:"s_query,omitempty"`
	AQuery *BooleanSearchQuery_ArbitraryBoolQuery   `protobuf:"bytes,4,opt,name=a_query" json:"a_query,omitempty"`
	DQuery *BooleanSearchQuery_DisjunctionQuery     `protobuf:"bytes,5,opt,name=d_query" json:"d_query,omitempty"`
}

func (m *BooleanSearchQuery) Reset()         { *m = BooleanSearchQuery{} }
func (m *BooleanSearchQuery) String() string { return proto.CompactTextString(m) }
func (*BooleanSearchQuery) ProtoMessage()    {}

func (m *BooleanSearchQuery) GetNQuery() *BooleanSearchQuery_NegatedConjunction {
	if m != nil {
		return m.NQuery
	}
	return nil
}

func (m *BooleanSearchQuery) GetSQuery() *BooleanSearchQuery_SearchableNormalForm {
	if m != nil {
		return m.SQuery
	}
	return nil
}

func (m *BooleanSearchQuery) GetAQuery() *BooleanSearchQuery_ArbitraryBoolQuery {
	if m != nil {
		return m.AQuery
	}
	return nil
}

func (m *BooleanSearchQuery) GetDQuery() *BooleanSearchQuery_DisjunctionQuery {
	if m != nil {
		return m.DQuery
	}
	return nil
}

type BooleanSearchQuery_NegatedConjunction struct {
	Stag      []byte `protobuf:"bytes,1,opt,name=stag,proto3" json:"stag,omitempty"`
	BoolQuery string `protobuf:"bytes,2,opt,name=bool_query" json:"bool_query,omitempty"`
}

func (m *BooleanSearchQuery_NegatedConjunction) Reset()         { *m = BooleanSearchQuery_NegatedConjunction{} }
func (m *BooleanSearchQuery_NegatedConjunction) String() string { return proto.CompactTextString(m) }
func (*BooleanSearchQuery_NegatedConjunction) ProtoMessage()    {}

type BooleanSearchQuery_SearchableNormalForm struct {
	// stag ^ (xtag ^ ...)
	Stag      []byte `protobuf:"bytes,1,opt,name=stag,proto3" json:"stag,omitempty"`
	BoolQuery string `protobuf:"bytes,2,opt,name=bool_query" json:"bool_query,omitempty"`
}

func (m *BooleanSearchQuery_SearchableNormalForm) Reset() {
	*m = BooleanSearchQuery_SearchableNormalForm{}
}
func (m *BooleanSearchQuery_SearchableNormalForm) String() string { return proto.CompactTextString(m) }
func (*BooleanSearchQuery_SearchableNormalForm) ProtoMessage()    {}

type BooleanSearchQuery_ArbitraryBoolQuery struct {
	// TRUE ^ (xtag AND xtag OR xtag)
	BoolQuery string `protobuf:"bytes,1,opt,name=bool_query" json:"bool_query,omitempty"`
}

func (m *BooleanSearchQuery_ArbitraryBoolQuery) Reset()         { *m = BooleanSearchQuery_ArbitraryBoolQuery{} }
func (m *BooleanSearchQuery_ArbitraryBoolQuery) String() string { return proto.CompactTextString(m) }
func (*BooleanSearchQuery_ArbitraryBoolQuery) ProtoMessage()    {}

type BooleanSearchQuery_DisjunctionQuery struct {
	BoolQuery string `protobuf:"bytes,1,opt,name=bool_query" json:"bool_query,omitempty"`
}

func (m *BooleanSearchQuery_DisjunctionQuery) Reset()         { *m = BooleanSearchQuery_DisjunctionQuery{} }
func (m *BooleanSearchQuery_DisjunctionQuery) String() string { return proto.CompactTextString(m) }
func (*BooleanSearchQuery_DisjunctionQuery) ProtoMessage()    {}

type XTokenRequest struct {
	// TODO(roasbeef): add a search ID?
	Stag     []byte `protobuf:"bytes,1,opt,name=stag,proto3" json:"stag,omitempty"`
	DocIndex int32  `protobuf:"varint,2,opt,name=doc_index" json:"doc_index,omitempty"`
}

func (m *XTokenRequest) Reset()         { *m = XTokenRequest{} }
func (m *XTokenRequest) String() string { return proto.CompactTextString(m) }
func (*XTokenRequest) ProtoMessage()    {}

type XTokenResponse struct {
	Stag     []byte `protobuf:"bytes,1,opt,name=stag,proto3" json:"stag,omitempty"`
	DocIndex int32  `protobuf:"varint,2,opt,name=doc_index" json:"doc_index,omitempty"`
	Xtoken   []byte `protobuf:"bytes,3,opt,name=xtoken,proto3" json:"xtoken,omitempty"`
}

func (m *XTokenResponse) Reset()         { *m = XTokenResponse{} }
func (m *XTokenResponse) String() string { return proto.CompactTextString(m) }
func (*XTokenResponse) ProtoMessage()    {}

type Error struct {
}

func (m *Error) Reset()         { *m = Error{} }
func (m *Error) String() string { return proto.CompactTextString(m) }
func (*Error) ProtoMessage()    {}

func init() {
	proto.RegisterEnum("sse_protos.IndexData_MessageType", IndexData_MessageType_name, IndexData_MessageType_value)
	proto.RegisterEnum("sse_protos.BooleanSearchQuery_SearchType", BooleanSearchQuery_SearchType_name, BooleanSearchQuery_SearchType_value)
}

// Client API for EncryptedSearch service

type EncryptedSearchClient interface {
	// TODO(roasbeef): Do the first two need returns?
	InitializeIndex(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_InitializeIndexClient, error)
	UploadCipherDocs(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_UploadCipherDocsClient, error)
	KeywordSearch(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_KeywordSearchClient, error)
	ConjunctiveSearchRequest(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_ConjunctiveSearchRequestClient, error)
	XTokenExchange(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_XTokenExchangeClient, error)
}

type encryptedSearchClient struct {
	cc *grpc.ClientConn
}

func NewEncryptedSearchClient(cc *grpc.ClientConn) EncryptedSearchClient {
	return &encryptedSearchClient{cc}
}

func (c *encryptedSearchClient) InitializeIndex(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_InitializeIndexClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[0], c.cc, "/sse_protos.EncryptedSearch/InitializeIndex", opts...)
	if err != nil {
		return nil, err
	}
	x := &encryptedSearchInitializeIndexClient{stream}
	return x, nil
}

type EncryptedSearch_InitializeIndexClient interface {
	Send(*IndexData) error
	CloseAndRecv() (*IndexAck, error)
	grpc.ClientStream
}

type encryptedSearchInitializeIndexClient struct {
	grpc.ClientStream
}

func (x *encryptedSearchInitializeIndexClient) Send(m *IndexData) error {
	return x.ClientStream.SendProto(m)
}

func (x *encryptedSearchInitializeIndexClient) CloseAndRecv() (*IndexAck, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(IndexAck)
	if err := x.ClientStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *encryptedSearchClient) UploadCipherDocs(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_UploadCipherDocsClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[1], c.cc, "/sse_protos.EncryptedSearch/UploadCipherDocs", opts...)
	if err != nil {
		return nil, err
	}
	x := &encryptedSearchUploadCipherDocsClient{stream}
	return x, nil
}

type EncryptedSearch_UploadCipherDocsClient interface {
	Send(*CipherDoc) error
	CloseAndRecv() (*CipherDocAck, error)
	grpc.ClientStream
}

type encryptedSearchUploadCipherDocsClient struct {
	grpc.ClientStream
}

func (x *encryptedSearchUploadCipherDocsClient) Send(m *CipherDoc) error {
	return x.ClientStream.SendProto(m)
}

func (x *encryptedSearchUploadCipherDocsClient) CloseAndRecv() (*CipherDocAck, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(CipherDocAck)
	if err := x.ClientStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *encryptedSearchClient) KeywordSearch(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_KeywordSearchClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[2], c.cc, "/sse_protos.EncryptedSearch/KeywordSearch", opts...)
	if err != nil {
		return nil, err
	}
	x := &encryptedSearchKeywordSearchClient{stream}
	return x, nil
}

type EncryptedSearch_KeywordSearchClient interface {
	Send(*KeywordQuery) error
	Recv() (*CipherDoc, error)
	grpc.ClientStream
}

type encryptedSearchKeywordSearchClient struct {
	grpc.ClientStream
}

func (x *encryptedSearchKeywordSearchClient) Send(m *KeywordQuery) error {
	return x.ClientStream.SendProto(m)
}

func (x *encryptedSearchKeywordSearchClient) Recv() (*CipherDoc, error) {
	m := new(CipherDoc)
	if err := x.ClientStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *encryptedSearchClient) ConjunctiveSearchRequest(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_ConjunctiveSearchRequestClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[3], c.cc, "/sse_protos.EncryptedSearch/ConjunctiveSearchRequest", opts...)
	if err != nil {
		return nil, err
	}
	x := &encryptedSearchConjunctiveSearchRequestClient{stream}
	return x, nil
}

type EncryptedSearch_ConjunctiveSearchRequestClient interface {
	Send(*BooleanSearchQuery) error
	Recv() (*CipherDoc, error)
	grpc.ClientStream
}

type encryptedSearchConjunctiveSearchRequestClient struct {
	grpc.ClientStream
}

func (x *encryptedSearchConjunctiveSearchRequestClient) Send(m *BooleanSearchQuery) error {
	return x.ClientStream.SendProto(m)
}

func (x *encryptedSearchConjunctiveSearchRequestClient) Recv() (*CipherDoc, error) {
	m := new(CipherDoc)
	if err := x.ClientStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *encryptedSearchClient) XTokenExchange(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_XTokenExchangeClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[4], c.cc, "/sse_protos.EncryptedSearch/XTokenExchange", opts...)
	if err != nil {
		return nil, err
	}
	x := &encryptedSearchXTokenExchangeClient{stream}
	return x, nil
}

type EncryptedSearch_XTokenExchangeClient interface {
	Send(*XTokenRequest) error
	Recv() (*XTokenResponse, error)
	grpc.ClientStream
}

type encryptedSearchXTokenExchangeClient struct {
	grpc.ClientStream
}

func (x *encryptedSearchXTokenExchangeClient) Send(m *XTokenRequest) error {
	return x.ClientStream.SendProto(m)
}

func (x *encryptedSearchXTokenExchangeClient) Recv() (*XTokenResponse, error) {
	m := new(XTokenResponse)
	if err := x.ClientStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server API for EncryptedSearch service

type EncryptedSearchServer interface {
	// TODO(roasbeef): Do the first two need returns?
	InitializeIndex(EncryptedSearch_InitializeIndexServer) error
	UploadCipherDocs(EncryptedSearch_UploadCipherDocsServer) error
	KeywordSearch(EncryptedSearch_KeywordSearchServer) error
	ConjunctiveSearchRequest(EncryptedSearch_ConjunctiveSearchRequestServer) error
	XTokenExchange(EncryptedSearch_XTokenExchangeServer) error
}

func RegisterEncryptedSearchServer(s *grpc.Server, srv EncryptedSearchServer) {
	s.RegisterService(&_EncryptedSearch_serviceDesc, srv)
}

func _EncryptedSearch_InitializeIndex_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EncryptedSearchServer).InitializeIndex(&encryptedSearchInitializeIndexServer{stream})
}

type EncryptedSearch_InitializeIndexServer interface {
	SendAndClose(*IndexAck) error
	Recv() (*IndexData, error)
	grpc.ServerStream
}

type encryptedSearchInitializeIndexServer struct {
	grpc.ServerStream
}

func (x *encryptedSearchInitializeIndexServer) SendAndClose(m *IndexAck) error {
	return x.ServerStream.SendProto(m)
}

func (x *encryptedSearchInitializeIndexServer) Recv() (*IndexData, error) {
	m := new(IndexData)
	if err := x.ServerStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _EncryptedSearch_UploadCipherDocs_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EncryptedSearchServer).UploadCipherDocs(&encryptedSearchUploadCipherDocsServer{stream})
}

type EncryptedSearch_UploadCipherDocsServer interface {
	SendAndClose(*CipherDocAck) error
	Recv() (*CipherDoc, error)
	grpc.ServerStream
}

type encryptedSearchUploadCipherDocsServer struct {
	grpc.ServerStream
}

func (x *encryptedSearchUploadCipherDocsServer) SendAndClose(m *CipherDocAck) error {
	return x.ServerStream.SendProto(m)
}

func (x *encryptedSearchUploadCipherDocsServer) Recv() (*CipherDoc, error) {
	m := new(CipherDoc)
	if err := x.ServerStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _EncryptedSearch_KeywordSearch_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EncryptedSearchServer).KeywordSearch(&encryptedSearchKeywordSearchServer{stream})
}

type EncryptedSearch_KeywordSearchServer interface {
	Send(*CipherDoc) error
	Recv() (*KeywordQuery, error)
	grpc.ServerStream
}

type encryptedSearchKeywordSearchServer struct {
	grpc.ServerStream
}

func (x *encryptedSearchKeywordSearchServer) Send(m *CipherDoc) error {
	return x.ServerStream.SendProto(m)
}

func (x *encryptedSearchKeywordSearchServer) Recv() (*KeywordQuery, error) {
	m := new(KeywordQuery)
	if err := x.ServerStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _EncryptedSearch_ConjunctiveSearchRequest_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EncryptedSearchServer).ConjunctiveSearchRequest(&encryptedSearchConjunctiveSearchRequestServer{stream})
}

type EncryptedSearch_ConjunctiveSearchRequestServer interface {
	Send(*CipherDoc) error
	Recv() (*BooleanSearchQuery, error)
	grpc.ServerStream
}

type encryptedSearchConjunctiveSearchRequestServer struct {
	grpc.ServerStream
}

func (x *encryptedSearchConjunctiveSearchRequestServer) Send(m *CipherDoc) error {
	return x.ServerStream.SendProto(m)
}

func (x *encryptedSearchConjunctiveSearchRequestServer) Recv() (*BooleanSearchQuery, error) {
	m := new(BooleanSearchQuery)
	if err := x.ServerStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _EncryptedSearch_XTokenExchange_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EncryptedSearchServer).XTokenExchange(&encryptedSearchXTokenExchangeServer{stream})
}

type EncryptedSearch_XTokenExchangeServer interface {
	Send(*XTokenResponse) error
	Recv() (*XTokenRequest, error)
	grpc.ServerStream
}

type encryptedSearchXTokenExchangeServer struct {
	grpc.ServerStream
}

func (x *encryptedSearchXTokenExchangeServer) Send(m *XTokenResponse) error {
	return x.ServerStream.SendProto(m)
}

func (x *encryptedSearchXTokenExchangeServer) Recv() (*XTokenRequest, error) {
	m := new(XTokenRequest)
	if err := x.ServerStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _EncryptedSearch_serviceDesc = grpc.ServiceDesc{
	ServiceName: "sse_protos.EncryptedSearch",
	HandlerType: (*EncryptedSearchServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "InitializeIndex",
			Handler:       _EncryptedSearch_InitializeIndex_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "UploadCipherDocs",
			Handler:       _EncryptedSearch_UploadCipherDocs_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "KeywordSearch",
			Handler:       _EncryptedSearch_KeywordSearch_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "ConjunctiveSearchRequest",
			Handler:       _EncryptedSearch_ConjunctiveSearchRequest_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "XTokenExchange",
			Handler:       _EncryptedSearch_XTokenExchange_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
}
