package link

type Config struct {
	Table       string
	UniqueIndex string

	LeftColumn      string
	LeftForeignKey  string
	LeftNotFoundErr error

	RightColumn      string
	RightForeignKey  string
	RightNotFoundErr error
}
