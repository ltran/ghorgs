package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/shurcooL/githubql"
	"golang.org/x/oauth2"
	git "gopkg.in/src-d/go-git.v4"
)

var (
	repoRoot string
	owner    string
)

type Repository struct {
	URL githubql.URI
}

type Repositories struct {
	Nodes []Repository
}

type query struct {
	RepositoryOwner struct {
		Repositories Repositories `graphql:"repositories(first: $repositoryFirst)"`
	} `graphql:"repositoryOwner(login: $repositoryLogin)"`
}

func main() {
	q := query{}
	variables := map[string]interface{}{
		"repositoryLogin": githubql.String("wrsinc"),
		"repositoryFirst": githubql.Int(100),
	}

	repos := FetchRepositories(client(), &q, variables)
	var wg sync.WaitGroup
	wg.Add(len(repos))

	cc := make(chan githubql.URI)

	go func() {
		for _, repo := range repos {
			cc <- repo.URL
		}
	}()

	for i := 0; i < len(repos); i++ {
		select {
		case u := <-cc:
			go func() {
				defer wg.Done()
				Clone(u)
			}()
		}
	}

	wg.Wait()
}

func client() *githubql.Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	return githubql.NewClient(httpClient)

}

func FetchRepositories(client *githubql.Client, q *query, vars map[string]interface{}) []Repository {
	err := client.Query(context.Background(), q, vars)
	if err != nil {
		fmt.Println(err)
	}

	return q.RepositoryOwner.Repositories.Nodes
}

func Clone(url githubql.URI) {
	x := strings.Split(url.String(), "/")
	_, err := git.PlainClone(repoRoot+"/"+x[len(x)-1], false, &git.CloneOptions{
		URL:      url.String(),
		Progress: os.Stdout,
	})

	if err != nil {
		fmt.Println(err)
	}
}

func init() {
	repoRoot = os.Getenv("REPO_PATH")
	if repoRoot == "" {
		repoRoot = "/tmp/foo"
	}
	owner = os.Getenv("ORG_NAME")
}
