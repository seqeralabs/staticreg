package serviceinfo

var (
	version string = "dev"
	commit  string
	date    string
)

type ServiceInfo struct {
	Version   string `json:"version"`
	CommitID  string `json:"commitId"`
	BuildTime string `json:"buildTime"`
}

func New() *ServiceInfo {
	commitTruncated := ""
	if len(commit) > 7 {
		commitTruncated = commit[:7]
	}
	sinfo := &ServiceInfo{
		Version:   version,
		CommitID:  commitTruncated,
		BuildTime: date,
	}

	return sinfo
}
