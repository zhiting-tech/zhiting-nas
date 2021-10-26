package session

type User struct {
	UserID     int
	Nickname   string
	IsOwner    bool
	ScopeToken string
	AreaId     int
	AreaName   string
}
