module github.com/abyssdigger/lgr

go 1.25.1

require github.com/stretchr/testify v1.11.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v0.1.0 // published without license file.
	v0.1.1 // contains only added license file and retractions.
	v0.1.2 // completely outdated api
	v0.2.0 // only retractions added (pkg.go.dev communication tested)
	v0.2.1 // removed retractions only (pkg.go.dev communication tested)
	v0.3.0 // dozen of bugs found, only minor api changes in the comparison with 0.4.*
	v0.4.0 // main type "logger" was private
	v0.4.1 // empty messages were processed, LogBytes/Log panicked on orphaned logger client
)