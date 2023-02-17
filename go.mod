module github.com/wader/bump

go 1.18

require (
	// bump: semver /github.com\/Masterminds\/semver\/v3 v(.*)/ git:https://github.com/Masterminds/semver|^3
	// bump: semver command go get -d github.com/Masterminds/semver/v3@v$LATEST && go mod tidy
	github.com/Masterminds/semver/v3 v3.2.0
	// bump: go-difflib /github.com\/pmezard\/go-difflib v(.*)/ git:https://github.com/pmezard/go-difflib|^1
	// bump: go-difflib command go get -d github.com/pmezard/go-difflib@v$LATEST && go mod tidy
	github.com/pmezard/go-difflib v1.0.0
)
