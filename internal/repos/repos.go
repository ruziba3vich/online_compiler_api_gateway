package repos

import "github.com/ruziba3vich/online_compiler_api_gateway/genprotos/genprotos/compiler_service"

type (
	Python interface {
		compiler_service.CodeExecutorClient
	}

	Java interface {
		compiler_service.CodeExecutorClient
	}

	Cpp interface {
		compiler_service.CodeExecutorClient
	}
)
