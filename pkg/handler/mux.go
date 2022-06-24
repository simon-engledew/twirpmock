package handler

import (
	"fmt"
	"github.com/brianvoe/gofakeit/v6"
	sproto "go.starlark.net/lib/proto"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"net/http"
	"time"
)

var predeclared = starlark.StringDict{
	"now": starlark.NewBuiltin("now", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		now := time.Now()

		var offset string
		if err := starlark.UnpackPositionalArgs("now", args, kwargs, 0, &offset); err != nil {
			return nil, err
		}

		if offset != "" {
			duration, err := time.ParseDuration(offset)
			if err != nil {
				return nil, err
			}
			now = now.Add(duration)
		}

		return timestampMessage(thread, now)
	}),
	"generate": starlark.NewBuiltin("generate", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var template string
		if err := starlark.UnpackPositionalArgs("generate", args, kwargs, 1, &template); err != nil {
			return nil, err
		}
		return starlark.String(thread.Local("faker").(*gofakeit.Faker).Generate(template)), nil
	}),
}

type ServeMux struct {
	*http.ServeMux
}

func NewServeMux() *ServeMux {
	return &ServeMux{
		ServeMux: http.NewServeMux(),
	}
}

func timestampMessage(thread *starlark.Thread, t time.Time) (*sproto.Message, error) {
	data, err := proto.Marshal(timestamppb.New(t))
	if err != nil {
		return nil, err
	}
	return sproto.Unmarshal(thread.Local("timestamp").(protoreflect.MessageDescriptor), data)
}

func jsonUnmarshal(desc protoreflect.MessageDescriptor, data []byte) (*sproto.Message, error) {
	out := dynamicpb.NewMessage(desc)
	err := protojson.Unmarshal(data, out)
	if err != nil {
		return nil, err
	}
	// work around limitations in the sproto package
	data, err = proto.Marshal(out)
	if err != nil {
		return nil, err
	}
	return sproto.Unmarshal(desc, data)
}

func exec(thread *starlark.Thread, method protoreflect.MethodDescriptor, filename string, src interface{}, input starlark.Value) ([]byte, error) {
	globals, err := starlark.ExecFile(thread, filename, src, predeclared)
	if err != nil {
		return nil, err
	}

	fn, ok := globals[string(method.Name())]
	if !ok {
		return nil, nil
	}

	zero, err := sproto.UnmarshalText(method.Output(), []byte(""))
	if err != nil {
		return nil, err
	}

	output, err := starlark.Call(thread, fn, starlark.Tuple{input, zero}, nil)
	if err != nil {
		return nil, err
	}

	data, err := protojson.Marshal(output.(*sproto.Message).Message())
	if err != nil {
		return nil, err
	}

	return data, err
}

func (h *ServeMux) Handle(set *descriptorpb.FileDescriptorSet, filename string, src interface{}) error {
	files, err := protodesc.NewFiles(set)
	if err != nil {
		return err
	}

	timestamp, err := files.FindDescriptorByName("google.protobuf.Timestamp")

	setup := func(fd protoreflect.FileDescriptor) error {
		for i := 0; i < fd.Services().Len(); i++ {
			service := fd.Services().Get(i)
			for j := 0; j < service.Methods().Len(); j++ {
				method := service.Methods().Get(j)

				path := fmt.Sprintf("/twirp/%s/%s", service.FullName(), method.Name())

				h.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					if err == nil {
						err = r.Body.Close()
					}
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					var unmarshal func(desc protoreflect.MessageDescriptor, data []byte) (*sproto.Message, error)

					switch r.Header.Get("Content-Type") {
					case "application/json":
						unmarshal = jsonUnmarshal
					case "application/protobuf":
						unmarshal = sproto.Unmarshal
					default:
						http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
						return
					}

					req, err := unmarshal(method.Input(), body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					thread := &starlark.Thread{Name: "ServeMux"}
					thread.SetLocal("faker", gofakeit.New(1))
					thread.SetLocal("timestamp", timestamp)

					data, err := exec(thread, method, filename, src, req)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					header := w.Header()
					header.Set("Content-Type", "application/json")
					w.WriteHeader(200)
					_, _ = w.Write(data)
				})
			}
		}

		return nil
	}

	files.RangeFiles(func(descriptor protoreflect.FileDescriptor) bool {
		err = setup(descriptor)
		return err == nil
	})

	return err
}
