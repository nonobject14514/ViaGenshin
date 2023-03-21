package core

import (
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/Jx2f/ViaGenshin/internal/mapper"
	"github.com/Jx2f/ViaGenshin/pkg/logger"
)

var (
	MarshalOptions = &protojson.MarshalOptions{
		AllowPartial:   true,
		UseEnumNumbers: true,
	}
	UnmarshalOptions = &protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	Marshaler = &jsonpb.Marshaler{
		EnumsAsInts: true,
	}
	Unmarshaler = &jsonpb.Unmarshaler{
		AllowUnknownFields: true,
	}
)

func (s *Session) ConvertPacket(from, to mapper.Protocol, fromCmd uint16, p []byte) ([]byte, error) {
	name := s.mapping.CommandNameMap[from][fromCmd]
	fromDesc := s.mapping.MessageDescMap[from][name]
	if fromDesc == nil {
		return p, fmt.Errorf("unknown from message %s in %s", name, to)
	}
	fromPacket := dynamic.NewMessage(fromDesc)
	if err := fromPacket.Unmarshal(p); err != nil {
		return p, err
	}
	fromJson, err := fromPacket.MarshalJSONPB(Marshaler)
	if err != nil {
		return p, err
	}
	toJson, err := s.HandlePacket(from, to, name, fromJson)
	if err != nil {
		return p, err
	}
	logger.Trace().RawJSON("from", fromJson).RawJSON("to", toJson).Msgf("Packet %s converted from %s to %s", name, from, to)
	toDesc := s.mapping.MessageDescMap[to][name]
	if toDesc == nil {
		return p, fmt.Errorf("unknown to message %s in %s", name, to)
	}
	toPacket := dynamic.NewMessage(toDesc)
	if err := toPacket.UnmarshalJSONPB(Unmarshaler, toJson); err != nil {
		return p, err
	}
	toJson, err = toPacket.MarshalJSONPB(Marshaler)
	if err != nil {
		return p, err
	}
	return toPacket.Marshal()
}

func (s *Session) ConvertPacketByName(from, to mapper.Protocol, name string, p []byte) ([]byte, error) {
	fromDesc := s.mapping.MessageDescMap[from][name]
	if fromDesc == nil {
		return p, fmt.Errorf("unknown from message %s in %s", name, to)
	}
	fromPacket := dynamic.NewMessage(fromDesc)
	if err := fromPacket.Unmarshal(p); err != nil {
		return p, err
	}
	fromJson, err := fromPacket.MarshalJSONPB(Marshaler)
	if err != nil {
		return p, err
	}
	toJson, err := s.HandlePacket(from, to, name, fromJson)
	if err != nil {
		return p, err
	}
	logger.Trace().RawJSON("from", fromJson).RawJSON("to", toJson).Msgf("Packet %s converted from %s to %s", name, from, to)
	toDesc := s.mapping.MessageDescMap[to][name]
	if toDesc == nil {
		return p, fmt.Errorf("unknown to message %s in %s", name, to)
	}
	toPacket := dynamic.NewMessage(toDesc)
	if err := toPacket.UnmarshalJSONPB(Unmarshaler, toJson); err != nil {
		return p, err
	}
	toJson, err = toPacket.MarshalJSONPB(Marshaler)
	if err != nil {
		return p, err
	}
	return toPacket.Marshal()
}
