// Code generated by protoc-gen-go.
// source: sse.proto
// DO NOT EDIT!

/*
Package sse_protos is a generated protocol buffer package.

It is generated from these files:
	sse.proto

It has these top-level messages:
	DocInfo
	EncryptedDocInfo
	MetaData
	MetaDataAck
	TSetFragment
	XSetFilter
	FilterAck
	TSetAck
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

type DocInfo struct {
	DocId uint32 `protobuf:"varint,1,opt,name=doc_id" json:"doc_id,omitempty"`
}

func (m *DocInfo) Reset()         { *m = DocInfo{} }
func (m *DocInfo) String() string { return proto.CompactTextString(m) }
func (*DocInfo) ProtoMessage()    {}

type EncryptedDocInfo struct {
	EncryptedId []byte `protobuf:"bytes,1,opt,name=encrypted_id,proto3" json:"encrypted_id,omitempty"`
}

func (m *EncryptedDocInfo) Reset()         { *m = EncryptedDocInfo{} }
func (m *EncryptedDocInfo) String() string { return proto.CompactTextString(m) }
func (*EncryptedDocInfo) ProtoMessage()    {}

type MetaData struct {
	NumEidBytes   int32 `protobuf:"varint,1,opt,name=num_eid_bytes" json:"num_eid_bytes,omitempty"`
	NumBlindBytes int32 `protobuf:"varint,2,opt,name=num_blind_bytes" json:"num_blind_bytes,omitempty"`
}

func (m *MetaData) Reset()         { *m = MetaData{} }
func (m *MetaData) String() string { return proto.CompactTextString(m) }
func (*MetaData) ProtoMessage()    {}

type MetaDataAck struct {
	Ack bool `protobuf:"varint,1,opt,name=ack" json:"ack,omitempty"`
}

func (m *MetaDataAck) Reset()         { *m = MetaDataAck{} }
func (m *MetaDataAck) String() string { return proto.CompactTextString(m) }
func (*MetaDataAck) ProtoMessage()    {}

type TSetFragment struct {
	// (b || L) -> (B || s_i) XOR K
	Bucket []byte `protobuf:"bytes,1,opt,name=bucket,proto3" json:"bucket,omitempty"`
	Label  []byte `protobuf:"bytes,2,opt,name=label,proto3" json:"label,omitempty"`
	Data   []byte `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
}

func (m *TSetFragment) Reset()         { *m = TSetFragment{} }
func (m *TSetFragment) String() string { return proto.CompactTextString(m) }
func (*TSetFragment) ProtoMessage()    {}

type XSetFilter struct {
	// filter of g^( F_p(K_x, w) * xind )
	BloomFilter []byte `protobuf:"bytes,1,opt,name=bloom_filter,proto3" json:"bloom_filter,omitempty"`
}

func (m *XSetFilter) Reset()         { *m = XSetFilter{} }
func (m *XSetFilter) String() string { return proto.CompactTextString(m) }
func (*XSetFilter) ProtoMessage()    {}

type FilterAck struct {
	Ack bool `protobuf:"varint,1,opt,name=ack" json:"ack,omitempty"`
}

func (m *FilterAck) Reset()         { *m = FilterAck{} }
func (m *FilterAck) String() string { return proto.CompactTextString(m) }
func (*FilterAck) ProtoMessage()    {}

type TSetAck struct {
	Ack bool `protobuf:"varint,1,opt,name=ack" json:"ack,omitempty"`
}

func (m *TSetAck) Reset()         { *m = TSetAck{} }
func (m *TSetAck) String() string { return proto.CompactTextString(m) }
func (*TSetAck) ProtoMessage()    {}

type CipherDoc struct {
	// CipherDoc ID.
	DocId        uint32 `protobuf:"varint,1,opt,name=doc_id" json:"doc_id,omitempty"`
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
	proto.RegisterEnum("sse_protos.BooleanSearchQuery_SearchType", BooleanSearchQuery_SearchType_name, BooleanSearchQuery_SearchType_value)
}

// Client API for EncryptedSearch service

type EncryptedSearchClient interface {
	UploadMetaData(ctx context.Context, in *MetaData, opts ...grpc.CallOption) (*MetaDataAck, error)
	UploadTSet(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_UploadTSetClient, error)
	UploadXSetFilter(ctx context.Context, in *XSetFilter, opts ...grpc.CallOption) (*FilterAck, error)
	UploadCipherDocs(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_UploadCipherDocsClient, error)
	KeywordSearch(ctx context.Context, in *KeywordQuery, opts ...grpc.CallOption) (EncryptedSearch_KeywordSearchClient, error)
	FetchDocuments(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_FetchDocumentsClient, error)
	ConjunctiveSearchRequest(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_ConjunctiveSearchRequestClient, error)
	XTokenExchange(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_XTokenExchangeClient, error)
}

type encryptedSearchClient struct {
	cc *grpc.ClientConn
}

func NewEncryptedSearchClient(cc *grpc.ClientConn) EncryptedSearchClient {
	return &encryptedSearchClient{cc}
}

func (c *encryptedSearchClient) UploadMetaData(ctx context.Context, in *MetaData, opts ...grpc.CallOption) (*MetaDataAck, error) {
	out := new(MetaDataAck)
	err := grpc.Invoke(ctx, "/sse_protos.EncryptedSearch/UploadMetaData", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *encryptedSearchClient) UploadTSet(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_UploadTSetClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[0], c.cc, "/sse_protos.EncryptedSearch/UploadTSet", opts...)
	if err != nil {
		return nil, err
	}
	x := &encryptedSearchUploadTSetClient{stream}
	return x, nil
}

type EncryptedSearch_UploadTSetClient interface {
	Send(*TSetFragment) error
	CloseAndRecv() (*TSetAck, error)
	grpc.ClientStream
}

type encryptedSearchUploadTSetClient struct {
	grpc.ClientStream
}

func (x *encryptedSearchUploadTSetClient) Send(m *TSetFragment) error {
	return x.ClientStream.SendProto(m)
}

func (x *encryptedSearchUploadTSetClient) CloseAndRecv() (*TSetAck, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(TSetAck)
	if err := x.ClientStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *encryptedSearchClient) UploadXSetFilter(ctx context.Context, in *XSetFilter, opts ...grpc.CallOption) (*FilterAck, error) {
	out := new(FilterAck)
	err := grpc.Invoke(ctx, "/sse_protos.EncryptedSearch/UploadXSetFilter", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
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

func (c *encryptedSearchClient) KeywordSearch(ctx context.Context, in *KeywordQuery, opts ...grpc.CallOption) (EncryptedSearch_KeywordSearchClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[2], c.cc, "/sse_protos.EncryptedSearch/KeywordSearch", opts...)
	if err != nil {
		return nil, err
	}
	x := &encryptedSearchKeywordSearchClient{stream}
	if err := x.ClientStream.SendProto(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type EncryptedSearch_KeywordSearchClient interface {
	Recv() (*EncryptedDocInfo, error)
	grpc.ClientStream
}

type encryptedSearchKeywordSearchClient struct {
	grpc.ClientStream
}

func (x *encryptedSearchKeywordSearchClient) Recv() (*EncryptedDocInfo, error) {
	m := new(EncryptedDocInfo)
	if err := x.ClientStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *encryptedSearchClient) FetchDocuments(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_FetchDocumentsClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[3], c.cc, "/sse_protos.EncryptedSearch/FetchDocuments", opts...)
	if err != nil {
		return nil, err
	}
	x := &encryptedSearchFetchDocumentsClient{stream}
	return x, nil
}

type EncryptedSearch_FetchDocumentsClient interface {
	Send(*DocInfo) error
	Recv() (*CipherDoc, error)
	grpc.ClientStream
}

type encryptedSearchFetchDocumentsClient struct {
	grpc.ClientStream
}

func (x *encryptedSearchFetchDocumentsClient) Send(m *DocInfo) error {
	return x.ClientStream.SendProto(m)
}

func (x *encryptedSearchFetchDocumentsClient) Recv() (*CipherDoc, error) {
	m := new(CipherDoc)
	if err := x.ClientStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *encryptedSearchClient) ConjunctiveSearchRequest(ctx context.Context, opts ...grpc.CallOption) (EncryptedSearch_ConjunctiveSearchRequestClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[4], c.cc, "/sse_protos.EncryptedSearch/ConjunctiveSearchRequest", opts...)
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
	stream, err := grpc.NewClientStream(ctx, &_EncryptedSearch_serviceDesc.Streams[5], c.cc, "/sse_protos.EncryptedSearch/XTokenExchange", opts...)
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
	UploadMetaData(context.Context, *MetaData) (*MetaDataAck, error)
	UploadTSet(EncryptedSearch_UploadTSetServer) error
	UploadXSetFilter(context.Context, *XSetFilter) (*FilterAck, error)
	UploadCipherDocs(EncryptedSearch_UploadCipherDocsServer) error
	KeywordSearch(*KeywordQuery, EncryptedSearch_KeywordSearchServer) error
	FetchDocuments(EncryptedSearch_FetchDocumentsServer) error
	ConjunctiveSearchRequest(EncryptedSearch_ConjunctiveSearchRequestServer) error
	XTokenExchange(EncryptedSearch_XTokenExchangeServer) error
}

func RegisterEncryptedSearchServer(s *grpc.Server, srv EncryptedSearchServer) {
	s.RegisterService(&_EncryptedSearch_serviceDesc, srv)
}

func _EncryptedSearch_UploadMetaData_Handler(srv interface{}, ctx context.Context, buf []byte) (proto.Message, error) {
	in := new(MetaData)
	if err := proto.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(EncryptedSearchServer).UploadMetaData(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _EncryptedSearch_UploadTSet_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EncryptedSearchServer).UploadTSet(&encryptedSearchUploadTSetServer{stream})
}

type EncryptedSearch_UploadTSetServer interface {
	SendAndClose(*TSetAck) error
	Recv() (*TSetFragment, error)
	grpc.ServerStream
}

type encryptedSearchUploadTSetServer struct {
	grpc.ServerStream
}

func (x *encryptedSearchUploadTSetServer) SendAndClose(m *TSetAck) error {
	return x.ServerStream.SendProto(m)
}

func (x *encryptedSearchUploadTSetServer) Recv() (*TSetFragment, error) {
	m := new(TSetFragment)
	if err := x.ServerStream.RecvProto(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _EncryptedSearch_UploadXSetFilter_Handler(srv interface{}, ctx context.Context, buf []byte) (proto.Message, error) {
	in := new(XSetFilter)
	if err := proto.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(EncryptedSearchServer).UploadXSetFilter(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
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
	m := new(KeywordQuery)
	if err := stream.RecvProto(m); err != nil {
		return err
	}
	return srv.(EncryptedSearchServer).KeywordSearch(m, &encryptedSearchKeywordSearchServer{stream})
}

type EncryptedSearch_KeywordSearchServer interface {
	Send(*EncryptedDocInfo) error
	grpc.ServerStream
}

type encryptedSearchKeywordSearchServer struct {
	grpc.ServerStream
}

func (x *encryptedSearchKeywordSearchServer) Send(m *EncryptedDocInfo) error {
	return x.ServerStream.SendProto(m)
}

func _EncryptedSearch_FetchDocuments_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EncryptedSearchServer).FetchDocuments(&encryptedSearchFetchDocumentsServer{stream})
}

type EncryptedSearch_FetchDocumentsServer interface {
	Send(*CipherDoc) error
	Recv() (*DocInfo, error)
	grpc.ServerStream
}

type encryptedSearchFetchDocumentsServer struct {
	grpc.ServerStream
}

func (x *encryptedSearchFetchDocumentsServer) Send(m *CipherDoc) error {
	return x.ServerStream.SendProto(m)
}

func (x *encryptedSearchFetchDocumentsServer) Recv() (*DocInfo, error) {
	m := new(DocInfo)
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
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UploadMetaData",
			Handler:    _EncryptedSearch_UploadMetaData_Handler,
		},
		{
			MethodName: "UploadXSetFilter",
			Handler:    _EncryptedSearch_UploadXSetFilter_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "UploadTSet",
			Handler:       _EncryptedSearch_UploadTSet_Handler,
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
		},
		{
			StreamName:    "FetchDocuments",
			Handler:       _EncryptedSearch_FetchDocuments_Handler,
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
