package main

import (
	"cmp"
	"context"
	"fmt"
	"html/template"
	"os"
	"slices"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const sponsorsOutput = `{{range .}}
<a href="{{.LinkURL}}">
	<img src="{{.AvatarURL}}" class="img github-avatar" alt="{{.Name}}">
</a>
{{end}}`

var sponsorsOutputTpl = template.Must(template.New("sponsorsOutputTpl").Parse(sponsorsOutput))

type sponsorInfo struct {
	Name      string
	Login     string
	AvatarURL string
	LinkURL   string
	Amount    int
}

// Sponsors that are not on GitHub.
var staticSponsors = []sponsorInfo{
	{
		Name:      "Kastelo, Inc.",
		Login:     "kastelo",
		AvatarURL: "https://avatars.githubusercontent.com/u/20482589?v=4",
		LinkURL:   "https://kastelo.net/",
		Amount:    10000,
	},
	{
		Name:      "Reef Solutions",
		Login:     "reefsol",
		AvatarURL: "https://static.wixstatic.com/media/c19c76_e1ee443d4c5e4e3197a25eec7a0a97e5.png/v1/fill/w_78,h_81,al_c,lg_1,q_85,enc_auto/c19c76_e1ee443d4c5e4e3197a25eec7a0a97e5.png",
		LinkURL:   "https://kastelo.net/",
		Amount:    10000,
	},
}

func main() {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewClient(httpClient)
	var query struct {
		Organization struct {
			Sponsors struct {
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
				Edges []struct {
					Node struct {
						User struct {
							Login     string
							Name      string
							AvatarURL string
						} `graphql:"... on User"`
						Organization struct {
							Login     string
							Name      string
							AvatarURL string
						} `graphql:"... on Organization"`
						Sponsorable struct {
							Sponsorship struct {
								Edges []struct {
									Node struct {
										IsActive bool
										Tier     struct {
											MonthlyPriceInCents int
										}
									}
								}
							} `graphql:"sponsorshipsAsSponsor(maintainerLogins:\"syncthing\", first:5)"`
						} `graphql:"... on Sponsorable"`
					}
				}
			} `graphql:"sponsors(first:100, after:$cursor)"`
		} `graphql:"organization(login:\"syncthing\")"`
	}
	vars := map[string]any{
		"cursor": (*githubv4.String)(nil),
	}

	sponsors := staticSponsors

	for {
		if err := client.Query(context.Background(), &query, vars); err != nil {
			fmt.Println(err)
			return
		}

		for _, sponsor := range query.Organization.Sponsors.Edges {
			for _, sponsorship := range sponsor.Node.Sponsorable.Sponsorship.Edges {
				if sponsorship.Node.Tier.MonthlyPriceInCents >= 100*100 {
					sponsors = append(sponsors, sponsorInfo{
						Name:      sponsor.Node.User.Name,
						Login:     sponsor.Node.User.Login,
						AvatarURL: sponsor.Node.User.AvatarURL,
						LinkURL:   fmt.Sprintf("https://github.com/%s/", sponsor.Node.User.Login),
						Amount:    sponsorship.Node.Tier.MonthlyPriceInCents / 100,
					})
				}
			}
		}

		if !query.Organization.Sponsors.PageInfo.HasNextPage {
			break
		}
		vars["cursor"] = githubv4.NewString(query.Organization.Sponsors.PageInfo.EndCursor)
	}

	slices.SortFunc(sponsors, func(i, j sponsorInfo) int {
		if i.Amount != j.Amount {
			return cmp.Compare(j.Amount, i.Amount)
		}
		return strings.Compare(i.Login, j.Login)
	})

	if err := sponsorsOutputTpl.Execute(os.Stdout, sponsors); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
