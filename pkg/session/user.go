package session

type User struct {
	UserID              int
	Nickname            string
	IsOwner             bool
	ScopeToken          string
	AreaId              int
	AreaName            string
	AreaType            int
	DepartmentBaseInfos []DepartmentBaseInfo
}

type DepartmentBaseInfo struct {
	DepartmentId int32
	Name         string
	Role         int
}
