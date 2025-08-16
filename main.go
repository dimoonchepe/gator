package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dimoonchepe/gator/internal/config"
	"github.com/dimoonchepe/gator/internal/database"
	"github.com/dimoonchepe/gator/internal/rss"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type state struct {
	db     *database.Queries
	config *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(state *state, command command) error {
	handler, ok := c.handlers[command.name]
	if !ok {
		return fmt.Errorf("unknown command: %s", command.name)
	}
	return handler(state, command)
}

func (c *commands) register(name string, handler func(*state, command) error) {
	c.handlers[name] = handler
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(s *state, cmd command) error {
	return func(s *state, cmd command) error {
		currentUserName := s.config.CurrentUserName
		user, err := s.db.GetUser(context.Background(), currentUserName)
		if err != nil {
			return fmt.Errorf("error getting user: %v", err)
		}
		return handler(s, cmd, user)
	}
}

func scrapeFeeds(state *state) error {
	nextFeed, err := state.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = state.db.MarkFeedFetched(context.Background(), nextFeed.ID)
	if err != nil {
		return err
	}

	fetchedFeed, err := rss.FetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return err
	}

	for _, item := range fetchedFeed.Channel.Item {
		publishedAt, err := time.Parse("Mon, 02 Jan 2006 15:04:05 +0000", item.PubDate)
		if err != nil {
			fmt.Println(fmt.Errorf("error parsing published date: %v", err))
			continue
		}
		err = state.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			Title:       item.Title,
			Description: sql.NullString{String: item.Description, Valid: true},
			Url:         item.Link,
			FeedID:      nextFeed.ID,
			PublishedAt: sql.NullTime{Time: publishedAt, Valid: true},
		})
		if err != nil {
			fmt.Println(fmt.Errorf("error creating post: %v", err))
		}
	}
	return nil
}

//  Handlers

func handlerLogin(state *state, command command) error {
	if len(command.args) == 0 {
		return errors.New("login handler expects a single argument, but none was provided")
	}
	username := command.args[0]
	_, err := state.db.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("User %v does not exist", username)
	}
	err = state.config.SetUser(username)
	if err != nil {
		return fmt.Errorf("error setting user: %v", err)
	}
	fmt.Println("User set successfully")
	return nil
}

func handlerRegister(state *state, command command) error {
	if len(command.args) != 1 {
		return errors.New("register handler expects a single argument, but none was provided")
	}
	username := command.args[0]
	user, err := state.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	})
	if err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}
	fmt.Println("User ", username, "successfully created:", user)

	err = state.config.SetUser(username)
	if err != nil {
		return fmt.Errorf("error setting user: %v", err)
	}
	fmt.Println("User set successfully")
	return nil
}

func handlerReset(state *state, command command) error {
	err := state.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error deleting users: %v", err)
	}
	fmt.Println("All users deleted successfully")
	return nil
}

func handlerUsers(state *state, command command) error {
	users, err := state.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}
	for _, user := range users {
		isCurrent := ""
		if user.Name == state.config.CurrentUserName {
			isCurrent = " (current)"
		}
		fmt.Println("*", user.Name+isCurrent)
	}
	return nil
}

func handlerAggregate(state *state, command command) error {
	if len(command.args) != 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}
	timeBetweenReqs, err := time.ParseDuration(command.args[0])
	if err != nil {
		return fmt.Errorf("invalid time format: %v", err)
	}
	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		scrapeFeeds(state)
	}
}

func handlerAddFeed(state *state, command command, user database.User) error {
	if len(command.args) != 2 {
		return fmt.Errorf("usage: addfeed <name> <url>")
	}

	name := command.args[0]
	url := command.args[1]

	feed_id, err := state.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:     uuid.New(),
		Name:   name,
		Url:    url,
		UserID: user.ID,
	})
	if err != nil {
		return fmt.Errorf("error adding feed: %v", err)
	}

	fmt.Println("Feed added successfully:", name, url, user.Name)

	feedFollow, err := state.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: feed_id,
	})
	if err != nil {
		return fmt.Errorf("error following feed: %v", err)
	}

	fmt.Println("Feed", feedFollow.FeedName, "followed by", feedFollow.UserName)

	return nil
}

func handlerFeeds(state *state, command command) error {
	if len(command.args) != 0 {
		return fmt.Errorf("usage: feeds")
	}

	feeds, err := state.db.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error getting feeds: %v", err)
	}

	fmt.Println("Feeds:")
	for _, feed := range feeds {
		userName := ""
		if feed.UserName.Valid {
			userName = feed.UserName.String
		}
		fmt.Printf("- %s (%s) by %s\n", feed.Name, feed.Url, userName)
	}

	return nil
}

func handlerFollow(state *state, command command, user database.User) error {
	if len(command.args) != 1 {
		return fmt.Errorf("usage: follow <URL>")
	}

	url := command.args[0]
	feed, err := state.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("error getting feed: %v", err)
	}

	feedFollow, err := state.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("error following feed: %v", err)
	}

	fmt.Println("Feed", feedFollow.FeedName, "followed by", feedFollow.UserName)
	return nil
}

func handlerUnfollow(state *state, command command, user database.User) error {
	if len(command.args) != 1 {
		return fmt.Errorf("usage: unfollow <URL>")
	}

	url := command.args[0]
	feed, err := state.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("error getting feed: %v", err)
	}

	err = state.db.UnfollowFeed(context.Background(), database.UnfollowFeedParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("error unfollowing feed: %v", err)
	}

	fmt.Println("Feed", feed.Name, "unfollowed by", user.Name)
	return nil
}

func handlerFollowing(state *state, command command, user database.User) error {
	if len(command.args) != 0 {
		return fmt.Errorf("usage: following")
	}

	feedFollows, err := state.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error getting feed follows: %v", err)
	}

	fmt.Println("User", user.Name, "following:")
	for _, feedFollow := range feedFollows {
		fmt.Printf("- '%s'\n", feedFollow.FeedName)
	}

	return nil
}

func handlerBrowse(state *state, command command, user database.User) error {
	if len(command.args) > 1 {
		return fmt.Errorf("usage: browse [<limit=2>]")
	}
	limit := 2
	if len(command.args) == 1 {
		l, err := strconv.Atoi(command.args[0])
		if err != nil {
			return fmt.Errorf("invalid limit: %v", err)
		}
		limit = l
	}

	posts, err := state.db.GetPostsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error getting posts: %v", err)
	}

	// print posts limited by limit
	fmt.Println("Posts:")
	for _, post := range posts[:limit] {
		fmt.Printf("- '%s'\n", post.Title)
	}

	return nil
}

// Main

func main() {
	// Read config file
	myState := &state{}
	conf := config.Read()
	myState.config = &conf
	dbURL := conf.DBURL
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	dbQueries := database.New(db)
	myState.db = dbQueries

	myCommands := &commands{
		handlers: make(map[string]func(*state, command) error),
	}
	myCommands.register("login", handlerLogin)
	myCommands.register("register", handlerRegister)
	myCommands.register("reset", handlerReset)
	myCommands.register("users", handlerUsers)
	myCommands.register("agg", handlerAggregate)
	myCommands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	myCommands.register("feeds", handlerFeeds)
	myCommands.register("follow", middlewareLoggedIn(handlerFollow))
	myCommands.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	myCommands.register("following", middlewareLoggedIn(handlerFollowing))
	myCommands.register("browse", middlewareLoggedIn(handlerBrowse))

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("No command provided")
		os.Exit(1)
	}
	command := command{
		name: args[0],
		args: args[1:],
	}
	err = myCommands.run(myState, command)
	if err != nil {
		fmt.Printf("Error running command: %v\n", err)
		os.Exit(1)
	}
}
