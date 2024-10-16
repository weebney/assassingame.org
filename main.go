package main

import (
	"fmt"
	"log"
	"net/mail"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./public"), false))
		return nil
	})

	// todo: game creation email
	// todo: game start handler

	// Elimination handler --- gets new target when eliminated
	app.OnRecordBeforeUpdateRequest("players").Add(func(e *core.RecordUpdateEvent) error {
		if e.Record.OriginalCopy().GetBool("is_alive") && !e.Record.GetBool("is_alive") {
			selfID := e.Record.GetString("user")
			gameID := e.Record.GetString("game")
			assassin, err := app.Dao().FindFirstRecordByFilter("players", fmt.Sprintf("target.id = '%s' && game.id = '%s'", selfID, gameID), nil)
			if err != nil {
				return err
			}
			targetID := e.Record.GetString("target")

			// winner handler
			if targetID == assassin.Id {
				gameRecord, err := app.Dao().FindFirstRecordByFilter("games", fmt.Sprintf("id = '%s'", gameID), nil)
				if err != nil {
					return err
				}
				gameRecord.Set("ended", true)
				gameRecord.Set("winner", assassin.Id)
				if err := app.Dao().SaveRecord(gameRecord); err != nil {
					return err
				}
			}

			assassin.Set("target", targetID)
			if err := app.Dao().SaveRecord(assassin); err != nil {
				return err
			}
		}
		return nil
	})

	// Target assigned emails
	app.OnRecordAfterUpdateRequest("players").Add(func(e *core.RecordUpdateEvent) error {
		if e.Record.OriginalCopy().GetString("target") != e.Record.GetString("target") && e.Record.GetString("target") != e.Record.GetString("user") {
			userRecord, err := app.Dao().FindFirstRecordByFilter("users", fmt.Sprintf("id = '%s'", e.Record.GetString("user")), nil)
			if err != nil {
				return err
			}

			gameId := e.Record.GetString("game")
			gameRecord, err := app.Dao().FindFirstRecordByFilter("games", fmt.Sprintf("id = '%s'", gameId), nil)
			if err != nil {
				return err
			}

			message := &mailer.Message{
				From: mail.Address{
					Address: app.Settings().Meta.SenderAddress,
					Name:    app.Settings().Meta.SenderName,
				},
				To:      []mail.Address{{Address: userRecord.Email()}},
				Subject: "New Target Assigned",
				HTML: fmt.Sprintf("You were assigned a new target in the game %s; <a href='https://assassingame.org/game.html?code=%s'>click here to log in and see who it is!</a>",
					gameRecord.GetString("name"), gameRecord.GetString("game_code")),
			}
			return app.NewMailClient().Send(message)
		}
		return nil
	})

	// Elimintaion confirmation emails
	app.OnRecordBeforeUpdateRequest("players").Add(func(e *core.RecordUpdateEvent) error {
		if !e.Record.OriginalCopy().GetBool("target_pending_elimination") && e.Record.GetBool("target_pending_elimination") {
			gameID := e.Record.GetString("game")
			targetID := e.Record.GetString("target")
			targetUserRecord, err := app.Dao().FindFirstRecordByFilter("users", fmt.Sprintf("id = '%s'", targetID), nil)
			if err != nil {
				return err
			}
			gameRecord, err := app.Dao().FindFirstRecordByFilter("games", fmt.Sprintf("id = '%s'", gameID), nil)
			if err != nil {
				return err
			}
			message := &mailer.Message{
				From: mail.Address{
					Address: app.Settings().Meta.SenderAddress,
					Name:    app.Settings().Meta.SenderName,
				},
				To:      []mail.Address{{Address: targetUserRecord.Email()}},
				Subject: "Were you eliminated?",
				HTML: fmt.Sprintf("Were you eliminated in the game %s? If so, <a href='https://assassingame.org/game.html?code=%s'>click here to log in and mark yourself as eliminated!</a>",
					gameRecord.GetString("name"), gameRecord.GetString("game_code")),
			}
			return app.NewMailClient().Send(message)
		}
		return nil
	})

	// Game start and end emails
	app.OnRecordBeforeUpdateRequest("games").Add(func(e *core.RecordUpdateEvent) error {
		htmlRendered := ""
		subjectRendered := ""
		if !e.Record.OriginalCopy().GetBool("ended") && e.Record.GetBool("ended") {
			winnerId := e.Record.GetString("winner")
			winnerRecord, err := app.Dao().FindFirstRecordByFilter("users", fmt.Sprintf("id = '%s'", winnerId), nil)
			if err != nil {
				return err
			}

			winnerName := winnerRecord.GetString("name")

			subjectRendered = "Someome won your game!"
			htmlRendered = fmt.Sprintf("Congratulations to %s, the winner of %s.",
				winnerName,
				e.Record.GetString("name"))
		} else if !e.Record.OriginalCopy().GetBool("started") && e.Record.GetBool("started") {
			subjectRendered = "Your game has started!"
			htmlRendered = fmt.Sprintf("Your game, %s, has started! Your first target will be emailed to you shortly.",
				e.Record.GetString("name"))
		}

		playerIds := e.Record.GetStringSlice("players")

		allPlayerEmails := []mail.Address{}
		for _, playerId := range playerIds {
			playerRecord, err := app.Dao().FindFirstRecordByFilter("users", fmt.Sprintf("id = '%s'", playerId), nil)
			if err != nil {
				return err
			}
			allPlayerEmails = append(allPlayerEmails, mail.Address{
				Address: playerRecord.Email(),
				Name:    playerRecord.GetString("name"),
			})
		}

		if htmlRendered == "" {
			return nil
		}

		message := &mailer.Message{
			From: mail.Address{
				Address: app.Settings().Meta.SenderAddress,
				Name:    app.Settings().Meta.SenderName,
			},
			Bcc:     allPlayerEmails,
			Subject: subjectRendered,
			HTML:    htmlRendered,
		}

		return app.NewMailClient().Send(message)
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
