package version

var (
	Version   = "dev"
	Commit    = "none"
	Date      = "unknown"
	BuildUser = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuildUser string `json:"build_user"`
}

func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		BuildUser: BuildUser,
	}
}

func String() string {
	return Version
}
