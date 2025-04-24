package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v63/github"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

/*
I want to be able to connect to a MySQL database and run a query to get the data I need.
I want to be able to connect to a Postgres database and run a query to get the data I need.
I want to be able to connec to a Postgres database and run a query to input the data that I need.
I want to be able to connec to the Github API and get the data I need.
*/

func init() {
	// loads .env file into environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or could not load it:", err)
	}
}

// write a function that connects to Github and returns data from a specific user and from a specific repo.
func loginToGithub() {
	// Add styling to logging
	var (
		Green  = "\033[32m"
		Reset  = "\033[0m"
		Yellow = "\033[33m"
	)

	fmt.Println("------------------------------------------------------------------------------------------------")
	println(Yellow + "Logging into Github..." + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")

	client := github.NewClient(nil).WithAuthToken(os.Getenv("gitHubAuth"))
	owner := os.Getenv("GITHUB_OWNER")
	repo := os.Getenv("GITHUB_REPO")
	repoData, _, err := client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Repository Name: %s\n", repoData.GetName())
	fmt.Printf("Repository Description: %s\n", repoData.GetDescription())
	branch := "master"
	_, _, err = client.Repositories.GetCommit(context.Background(), owner, repo, branch, nil)
	if err != nil {
		log.Fatal(err)
	}

	println(Green + "Logged into Github" + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")
	// fmt.Printf("Last commit on main branch:\n")
	// 	fmt.Printf("Last Full commit message on main branch: %s\n", commit.GetCommit().GetMessage())
	// 	fmt.Printf("Author: %s\n", commit.GetCommit().GetAuthor().GetName())
	// 	fmt.Printf("Date: %s\n", commit.GetCommit().GetAuthor().GetDate())
	// 	fmt.Printf("ID: %d\n", repoData.GetID())
	// 	fmt.Printf("repo: %+v\n", repoData.GetFullName())
	// 	fmt.Printf("Owner: %v\n", repoData.GetOwner())
	// 	fmt.Printf("UpdatedAt: %v\n", repoData.GetUpdatedAt())
	// 	fmt.Printf("SHA: %s\n", commit.GetSHA())
	// 	fmt.Printf("PushedAt: %v\n", repoData.GetPushedAt())
	// 	Create a code break
	// 	fmt.Println("------------------------------------------------------------------------------------------------")
	// 	fmt.Printf("Size: %d\n", repoData.GetSize())
	// 	fmt.Printf("CommitsURL: %s\n", repoData.GetCommitsURL())
	// 	fmt.Printf("FullName: %s\n", repoData.GetFullName())
	// 	fmt.Printf("Name: %s\n", repoData.GetName())
	// 	fmt.Printf("Description: %s\n", repoData.GetDescription())
	// 	fmt.Printf("BranchesURL: %s\n", repoData.GetBranchesURL())
	// 	fmt.Printf("CreatedAt: %v\n", repoData.GetCreatedAt())
	// 	fmt.Printf("URL: %s\n", repoData.GetURL())
	// 	fmt.Println("Logged into Github")
}

func main() {

	var (
		Reset   = "\033[0m"
		Magenta = "\033[35m"
	)

	fmt.Println(Magenta + "------------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(Magenta + "            _____            _____                         _____          " + Reset)
	fmt.Println(Magenta + "           /\\    \\         /\\    \\                       /\\    \\         " + Reset)
	fmt.Println(Magenta + "          /::\\____\\       /::\\    \\                     /::\\    \\        " + Reset)
	fmt.Println(Magenta + "         /:::/    /       /::::\\    \\                   /::::\\    \\       " + Reset)
	fmt.Println(Magenta + "        /:::/    /       /::::::\\    \\                 /::::::\\    \\      " + Reset)
	fmt.Println(Magenta + "       /:::/    /       /:::/\\:::\\    \\               /:::/\\:::\\    \\     " + Reset)
	fmt.Println(Magenta + "      /:::/    /       /:::/  \\:::\\    \\             /:::/__\\:::\\    \\    " + Reset)
	fmt.Println(Magenta + "     /:::/    /       /:::/    \\:::\\    \\           /::::\\   \\:::\\    \\   " + Reset)
	fmt.Println(Magenta + "    /:::/    /       /:::/    / \\:::\\    \\         /::::::\\   \\:::\\    \\  " + Reset)
	fmt.Println(Magenta + "   \\:::/    /        /:::/    /   \\:::\\    \\      /:::/\\:::\\   \\:::\\____\\ " + Reset)
	fmt.Println(Magenta + "    \\:::/__/         /:::/____/     \\:::\\____\\    /:::/  \\:::\\   \\:::|    |" + Reset)
	fmt.Println(Magenta + "     \\:::\\   \\       \\:::\\    \\      \\  /     /  /:::/   |::::\\  /:::|____|" + Reset)
	fmt.Println(Magenta + "      \\:::\\   \\       \\:::\\    \\      \\/_____/  /___/    |:::::\\/:::/    / " + Reset)
	fmt.Println(Magenta + "       \\:::\\   \\       \\:::\\    \\                        |:::::::::/    /  " + Reset)
	fmt.Println(Magenta + "        \\:::\\   \\       \\:::\\    \\                       |::|\\::::/    /   " + Reset)
	fmt.Println(Magenta + "         \\:::\\   \\       \\:::\\    \\                      |::| \\::/____/    " + Reset)
	fmt.Println(Magenta + "          \\:::\\   \\       \\:::\\    \\                     |::|  ~|          " + Reset)
	fmt.Println(Magenta + "           \\:::\\   \\       \\:::\\    \\                    |::|   |          " + Reset)
	fmt.Println(Magenta + "            \\:::\\___\\       \\:::\\____\\                   \\::|   |          " + Reset)
	fmt.Println(Magenta + "             \\::/    /        \\::/    /                    \\:|   |          " + Reset)
	fmt.Println(Magenta + "              \\/____/ocal      \\/____/ontainer              \\|___|egistry          " + Reset)
	fmt.Println(Magenta + "------------------------------------------------------------------------------------------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(Magenta+" |", "                Commit SHA                 |            ", "PR Description            |", "  Image ID   | ", "  Image Size   | ", "  Image Tag   |"+Reset)
	fmt.Println(Magenta + "------------------------------------------------------------------------------------------------------------------------------------------------------------------------------" + Reset)

	// TODO: [Tabs] - [Github] List the Github Commit SHA
	// TODO: [Tabs] - [Github] List the Github PR-Description
	// TODO: [Tabs] - [Docker] List The Docker Image IDs
	// TODO: [Tabs] - [Docker] List The Docker Image Size
	// TODO: [Tabs] - [Docker] List The Docker Image Tags(If available)
	// TODO: [Tabs] - [Docker] Delete The Docker Image
	// TODO: [Tabs] - [Docker] Delete The Docker Container
	// TABS: [Tabs] - [Deployment] - Pull
	// TABS: [Tabs] - [Deployment] - List
	// TABS: [Tabs] - [Deployment] - Push
	// TABS: [Tabs] - [Deployment] - Delete
	loginToGithub()
}
