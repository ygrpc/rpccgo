package generator

import (
	"fmt"
	"path"
)

func AttachMessageFileFamilyPlan(file *FilePlan) {
	if file == nil {
		return
	}
	for i := range file.Services {
		file.Services[i].MessageFileFamily = BuildMessageFileFamilyPlan(*file, file.Services[i])
	}
}

func BuildMessageFileFamilyPlan(file FilePlan, service ServicePlan) MessageFileFamilyPlan {
	serviceName := lowerSnakeCase(service.GoName)
	prefix := file.GeneratedFilenamePrefix
	cgoPrefix := path.Join(path.Dir(prefix), cgoDirForFilePlan(file), path.Base(prefix))

	return MessageFileFamilyPlan{
		Runtime: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.runtime.rpccgo.go", prefix, serviceName),
			Enabled:  true,
		},
		MessageServer: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.server.message.rpccgo.go", prefix, serviceName),
			Enabled:  needsCGOMessageServerAdapter(service),
		},
		CGOMessageServer: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.server.message.cgo.rpccgo.go", cgoPrefix, serviceName),
			Enabled:  needsCGOMessageServerAdapter(service),
		},
		CGOMessageClient: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.client.message.cgo.rpccgo.go", cgoPrefix, serviceName),
			Enabled:  true,
		},
		ConnectServer: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.server.connect.rpccgo.go", prefix, serviceName),
			Enabled:  false,
		},
		GRPCServer: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.server.grpc.rpccgo.go", prefix, serviceName),
			Enabled:  false,
		},
		ConnectRemote: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.remote.connect.rpccgo.go", prefix, serviceName),
			Enabled:  false,
		},
		GRPCRemote: GeneratedFilePlan{
			Filename: fmt.Sprintf("%s.%s.remote.grpc.rpccgo.go", prefix, serviceName),
			Enabled:  false,
		},
	}
}

func BuildCodecFilePlan(file FilePlan, service ServicePlan) GeneratedFilePlan {
	return GeneratedFilePlan{
		Filename: fmt.Sprintf("%s.%s.codec.rpccgo.go", file.GeneratedFilenamePrefix, lowerSnakeCase(service.GoName)),
		Enabled:  service.NeedsCodec,
	}
}

func needsCGOMessageServerAdapter(service ServicePlan) bool {
	return service.Adapters.Has(AdapterTokenMessageConnect) || service.Adapters.Has(AdapterTokenMessageGRPC)
}
