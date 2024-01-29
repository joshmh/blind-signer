package proto

import (
	"google.golang.org/protobuf/proto"
)

func Marshal(dataType string, dataValue string) ([]byte, error) {
	data := &Data{
		Type:  dataType,
		Value: dataValue}

	return proto.Marshal(data)
}

func Unmarshal(bytes []byte) (string, string, error) {
	data := &Data{}
	err := proto.Unmarshal(bytes, data)
	if err != nil {
		return "", "", err

	}

	return data.GetType(), data.GetValue(), nil
}
