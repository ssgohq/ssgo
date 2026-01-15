package {{.Package}}

import (
	"context"

	"{{.Module}}/internal/svc"
	// Import types from kitex_gen (uncomment as needed)
	// "{{.TypesModule}}/kitex_gen/{{.ServiceLower}}"
)

// {{.Service}}Logic contains the business logic for {{.Service}}
type {{.Service}}Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// New{{.Service}}Logic creates a new {{.Service}}Logic
func New{{.Service}}Logic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.Service}}Logic {
	return &{{.Service}}Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// TODO: Implement your business logic methods here
// Types are imported from: {{.TypesModule}}/kitex_gen/{{.ServiceLower}}
//
// Example:
// func (l *{{.Service}}Logic) GetUser(req *{{.ServiceLower}}.GetUserRequest) (*{{.ServiceLower}}.GetUserResponse, error) {
//     // Access dependencies via l.svcCtx
//     // user, err := l.svcCtx.DB.GetUser(l.ctx, req.Id)
//     // if err != nil {
//     //     return nil, err
//     // }
//     // return &{{.ServiceLower}}.GetUserResponse{
//     //     Id:    user.ID,
//     //     Name:  user.Name,
//     //     Email: user.Email,
//     // }, nil
//     return &{{.ServiceLower}}.GetUserResponse{}, nil
// }