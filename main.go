package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	_ "github.com/ziutek/mymysql/godrv"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/User", postUserHandler).Methods("POST")
	r.HandleFunc("/User/{teamID:[0-9]+}", getUserHandler).Methods("GET")
	r.HandleFunc("/Subscribe", postSubscriptionHandler).Methods("POST")

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)

	//https://statsapi.web.nhl.com/api/v1/schedule?startDate=2016-04-16&endDate=2016-04-21
}

func postSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tSubscription := subscription{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		//TODO: Find out return error coce
		fmt.Println(err)
	}

	json.Unmarshal(body, &tSubscription)

	fmt.Println(tSubscription)

	insertSubscription(&tSubscription)

}

func insertSubscription(vSubscription *subscription) {
	//TODO: Put this somewhere else.
	db, err := sql.Open("mymysql", "sportsupdate/root/aiwojefoa39j2a9VVA3jj32fa3")
	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	sqlQuery := "SELECT `Users`.`UserId` FROM `ScoreBot`.`Users` WHERE `Users`.`UserName` = ?"

	row, err := db.Query(sqlQuery, vSubscription.Username)
	if err != nil {
		panic(err)
	}

	var userID int

	for row.Next() {

		err = row.Scan(&userID)
	}

	stmNewOutbox, err := db.Prepare("INSERT INTO `ScoreBot`.`Subscription` (`Users_UserId`, `Teams_TeamId`) VALUES (?, ?);")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmNewOutbox.Exec(userID, vSubscription.TeamID)
	if err != nil {
		panic(err.Error())
	}

}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	teamID := vars["teamID"]

	iTeamID, _ := strconv.Atoi(teamID)
	users := getUser(iTeamID)
	jsonUsers, _ := json.Marshal(users)

	w.Write(jsonUsers)
	tUser := user{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		//TODO: Find out return error coce
		fmt.Println(err)
	}

	fmt.Println(body)

	json.Unmarshal(body, &tUser)
}

func postUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tUser := user{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		//TODO: Find out return error coce
		fmt.Println(err)
	}

	json.Unmarshal(body, &tUser)

	insertUser(&tUser)

	// jsonUser, _ := json.Marshal(tUser)
	// w.Write(jsonUser)

}

func parseSchedule() {

	//TODO: Put this somewhere else.
	db, err := sql.Open("mysql", "root:password@/ScoreBot")
	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	//TODO: Select the entire season
	resp, err := http.Get("https://statsapi.web.nhl.com/api/v1/schedule?startDate=2016-04-16&endDate=2016-04-21")

	if err != nil {
		fmt.Println(err)
	}

	schedule := new(schedule)
	err = json.NewDecoder(resp.Body).Decode(schedule)

	if err != nil {
		fmt.Println(err)
	}

	insertMessageToSchedule(db, schedule)

}

func getUser(teamID int) []user {
	//TODO: Put this somewhere else.
	db, err := sql.Open("mysql", "root:password@/ScoreBot")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	fmt.Println("got a get")

	sqlQuery := "select UserName, Platform, Phone, Country, Joined from users where userId in (select Users_UserId from subscription where Teams_TeamId = ?)"

	row, err := db.Query(sqlQuery, teamID)
	if err != nil {
		panic(err)
	}

	var userList []user

	for row.Next() {
		u := user{}

		err = row.Scan(&u.Username, &u.Platform, &u.Phone, &u.Country, &u.Joined)

		userList = append(userList, u)
	}

	return userList
}

func insertUser(vUser *user) {
	//TODO: Put this somewhere else.
	db, err := sql.Open("mysql", "root:password@/ScoreBot")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	fmt.Println("got in the insert")

	stmNewOutbox, err := db.Prepare("INSERT INTO `ScoreBot`.`Users` (`UserName`, `Platform`, `Phone`, `Country`, `Joined`) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		panic(err.Error())
	}

	t := time.Now()

	_, err = stmNewOutbox.Exec(vUser.Username, vUser.Platform, vUser.Phone, vUser.Country, t)
	if err != nil {
		panic(err.Error())
	}

}

func insertMessageToSchedule(db *sql.DB, schedule *schedule) {

	for i := 0; i < len(schedule.Dates); i++ {

		for j := 0; j < len(schedule.Dates[i].Games); j++ {

			// stmNewOutbox, err := db.Prepare("INSERT INTO `ScoreBot`.`Event` (`Type`,`Media`,`MatchId`,`Score`, `IsSent`) VALUES (?, ?, ?, ?, 0)")
			stmNewOutbox, err := db.Prepare("INSERT INTO `ScoreBot`.`Games` (`AwayId`, `Start`,`Finish`,`HomeScore`,`AwayScore`,`Status`,`homeId`, `url`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
			if err != nil {
				panic(err.Error())
			}

			game := schedule.Dates[i].Games[j]

			defer stmNewOutbox.Close()
			gameDate, _ := time.Parse("2006-01-02T15:04:05Z07:00", game.GameDate)

			_, err = stmNewOutbox.Exec(game.Teams.Away.Team.ID, gameDate, gameDate, 0, 0, game.Status.DetailedState, game.Teams.Home.Team.ID, game.Link)
			if err != nil {
				panic(err.Error())
			}
		}

	}
}

type user struct {
	Username string
	Platform string
	Phone    string
	Country  string
	Joined   string
}

type subscription struct {
	Username string
	TeamID   int
}

type schedule struct {
	Copyright string `json:"copyright"`
	Dates     []struct {
		Date       string `json:"date"`
		Games      []game `json:"games"`
		TotalItems int    `json:"totalItems"`
	} `json:"dates"`
	TotalItems int `json:"totalItems"`
	Wait       int `json:"wait"`
}

type game struct {
	Content struct {
		Link string `json:"link"`
	} `json:"content"`
	GameDate string `json:"gameDate"`
	GamePk   int    `json:"gamePk"`
	GameType string `json:"gameType"`
	Link     string `json:"link"`
	Season   string `json:"season"`
	Status   struct {
		AbstractGameState string `json:"abstractGameState"`
		CodedGameState    string `json:"codedGameState"`
		DetailedState     string `json:"detailedState"`
		StatusCode        string `json:"statusCode"`
	} `json:"status"`
	Teams struct {
		Away struct {
			LeagueRecord struct {
				Losses int    `json:"losses"`
				Type   string `json:"type"`
				Wins   int    `json:"wins"`
			} `json:"leagueRecord"`
			Score int `json:"score"`
			Team  struct {
				ID   int    `json:"id"`
				Link string `json:"link"`
				Name string `json:"name"`
			} `json:"team"`
		} `json:"away"`
		Home struct {
			LeagueRecord struct {
				Losses int    `json:"losses"`
				Type   string `json:"type"`
				Wins   int    `json:"wins"`
			} `json:"leagueRecord"`
			Score int `json:"score"`
			Team  struct {
				ID   int    `json:"id"`
				Link string `json:"link"`
				Name string `json:"name"`
			} `json:"team"`
		} `json:"home"`
	} `json:"teams"`
	Venue struct {
		Name string `json:"name"`
	} `json:"venue"`
}
