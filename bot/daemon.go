package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"
)

const (
	// how often to confirm that users' sessions are still logged in, so they can be told to login again before they miss
	// out on codes, rather than after
	SessionCheckInterval = 24 * time.Hour
	// max sessions to check per processing loop, which spreads the checks out over time
	sessionChecksPerLoop = 50

	sessionExpiredMessage = "Looks like your SHiFT session has expired, so I can't redeem codes for you anymore " + X + "\n\n" +
		"Please login again with `/" + LOGIN + "`, and I'll pick up right where I left off!"
	link2KAccountMessage = "SHiFT won't let me redeem codes for you until you link your 2K account " + X + "\n\n" +
		"Once you've linked it on the SHiFT website, I'll automatically redeem any codes you missed (as long as they haven't expired)!"
	shiftErrorsMessage = "It seems like the last 5 code attempts I tried for you returned errors...\n" +
		"Your user credentials might be expired?\n\n" +
		"Maybe try logging in again with `/login`, but if this continues, please reach out on the [Official Discord Server](" + ServerLink + ")"
)

// errLink2KAccount is returned when SHiFT won't redeem codes until the user links their 2K account
var errLink2KAccount = errors.New(shift.LINK2K)

func (bot *Bot) StartUserRedemptionProcessing(interval time.Duration, stop <-chan bool) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-stop:
			ticker.Stop()
			slog.Info("User code redemption processing stopped")
			return

			// TODO add debouncing so we don't constantly trigger reprocessing if multiple codes come through close together
		case userID := <-bot.redemptionTrigger:
			// if we aren't triggering the code redemption processing for a specific user, then reset the top
			// control flow's interval so we don't run it back-to-back for all users
			if userID == "" {
				ticker.Reset(interval)
			}
			slog.Info("Started user code redemption processing from external trigger")
			bot.userRedemptionLoop(userID)

		case <-ticker.C:
			// check sessions first, so users with expired sessions (or errors that weren't from their session) are
			// handled accordingly before redemption
			bot.sessionCheckLoop()
			slog.Info("Started user code redemption processing")
			bot.userRedemptionLoop("")
		}
	}
}

func (bot *Bot) userRedemptionLoop(userID string) {
	var userCookies []store.UserCookies
	var err error
	// if a userID was provided, only get the cookies for that user
	if userID != "" {
		cookies, err := bot.storage.GetDecryptedUserCookies(userID)
		if err != nil {
			slog.Error("Failed to get cookies for user", "user_id", userID, "error", err.Error())
			return
		}
		userCookies = []store.UserCookies{
			store.UserCookies{
				UserID:  userID,
				Cookies: cookies,
			},
		}
		slog.Info("Retrieved decrypted user cookies for specific user", "user_id", userID)
	} else {
		userCookies, err = bot.storage.GetAllDecryptedUserCookiesSorted(-1)
		if err != nil {
			slog.Error("Error getting cookies", "error", err.Error())
			return
		}
		slog.Info("Retrieved decrypted user cookies", "count", len(userCookies))
	}

	for _, user := range userCookies {
		bot.redeemCodesForUser(user)
	}
}

// recoverUserPanic recovers and logs a panic, so one user can't stop processing for everyone else (or crash the bot).
// It must be deferred
func recoverUserPanic(userID string) {
	if r := recover(); r != nil {
		slog.Error("Recovered from panic while processing user", "user_id", userID, "panic", r, "stack", string(debug.Stack()))
	}
}

// redeemCodesForUser redeems any valid codes the user hasn't redeemed yet
func (bot *Bot) redeemCodesForUser(user store.UserCookies) {
	defer recoverUserPanic(user.UserID)

	platform, dm, err := bot.storage.GetUserPlatformAndDM(user.UserID)
	if err != nil {
		slog.Error("Error getting platform", "user_id", user.UserID, "error", err.Error())
		return
	}
	if platform == "" {
		slog.Debug("Skipping user with no platform set", "user_id", user.UserID)
		return
	}
	alert, err := bot.storage.GetUserAlert(user.UserID)
	if err != nil {
		slog.Error("Error getting alert", "user_id", user.UserID, "error", err.Error())
		return
	}
	// there's no point trying until they login again
	if alert == store.AlertSessionExpired {
		slog.Debug("Skipping user with expired session", "user_id", user.UserID)
		return
	}
	shiftErrors, err := bot.storage.GetShiftErrors(user.UserID)
	if err != nil {
		slog.Error("Error getting shift errors", "user_id", user.UserID, "error", err.Error())
		return
	}
	if len(shiftErrors) > 4 {
		bot.alertUser(user.UserID, dm, store.AlertShiftErrors, shiftErrorsMessage)
		return
	}
	codes, err := bot.storage.GetValidCodesNotRedeemedForUser(user.UserID, platform, 10)
	if err != nil {
		slog.Error("Error getting codes", "error", err.Error())
		return
	}
	slog.Debug("Retrieved unredeemed codes", "user_id", user.UserID, "codes", len(codes))

	client, err := bot.newShiftClient(user.Cookies)
	if err != nil {
		slog.Error("Error creating shift client", "user_id", user.UserID, "error", err.Error())
		return
	}

	for _, code := range codes {
		reward, status, err := bot.redeemCode(client, user, code, shift.Platform(platform))
		// these apply to every code, so stop until they're resolved (without counting them as errors)
		if errors.Is(err, shift.ErrNotLoggedIn) {
			bot.alertUser(user.UserID, dm, store.AlertSessionExpired, sessionExpiredMessage)
			return
		}
		if errors.Is(err, errLink2KAccount) {
			bot.alertUser(user.UserID, dm, store.AlertLink2KAccount, link2KAccountMessage)
			return
		}
		success := status == shift.SUCCESS
		if err != nil {
			slog.Error("Error redeeming code", "user_id", user.UserID, "code", code, "platform", platform, "error", err.Error())
			err2 := bot.storage.AddShiftError(user.UserID, code, platform, err.Error())
			if err2 != nil {
				slog.Error("Error adding shift error to db", "user_id", user.UserID, "code", code, "platform", platform, "error", err2.Error())
			}
		} else {
			// if no error was reported, then clear errors for this user
			// (for now, we treat them as only important if they're sequential)
			err = bot.storage.ClearShiftErrors(user.UserID)
			if err != nil {
				slog.Error("Error clearing shift errors from db", "user_id", user.UserID, "error", err.Error())
			}
			// redemption is working again, so whatever the user was alerted about has been resolved
			if alert != "" {
				err = bot.storage.ClearUserAlert(user.UserID)
				if err != nil {
					slog.Error("Error clearing alert", "user_id", user.UserID, "error", err.Error())
				}
				alert = ""
			}
			if reward != nil {
				set, err := bot.storage.SetCodeRewardAndSuccess(code, reward.Title, success)
				if err != nil {
					slog.Error("Error setting code reward", "code", code, "reward", reward.Title, "error", err.Error())
				} else if set {
					slog.Info("Set reward", "code", code, "reward", reward.Title)
				}
			}
		}
		if success && dm {
			str := Cheer + " I successfully redeemed `" + code + "` for you! " + Cheer + "\n\n"
			if reward != nil {
				str += "Looks like the prize was: `" + reward.Title + "`\n"
			}
			err = bot.DMUser(user.UserID, str)
			if err != nil {
				slog.Error("Error DMing user", "user_id", user.UserID, "error", err.Error())
			} else {
				slog.Info("DMed user", "user_id", user.UserID)
			}
		}
	}
}

// alertUser notifies the user about a problem (if they have DMs enabled), unless they've already been notified about it
func (bot *Bot) alertUser(userID string, dm bool, alert, message string) {
	changed, err := bot.storage.SetUserAlert(userID, alert)
	if err != nil {
		slog.Error("Error setting alert", "user_id", userID, "alert", alert, "error", err.Error())
		return
	}
	if !changed {
		return
	}
	slog.Info("Set alert", "user_id", userID, "alert", alert)
	if dm {
		err = bot.DMUser(userID, message)
		if err != nil {
			slog.Error("Failed to DM user", "user_id", userID, "alert", alert, "error", err.Error())
		} else {
			slog.Info("DMed user alert", "user_id", userID, "alert", alert)
		}
	}
}

// sessionCheckLoop checks sessions that haven't been verified recently. Otherwise, when there aren't new codes, an
// expired session would go unnoticed until the user had already missed out on codes
func (bot *Bot) sessionCheckLoop() {
	verifiedBefore := time.Now().Add(-SessionCheckInterval).Unix()
	users, err := bot.storage.GetDecryptedUserCookiesToVerify(verifiedBefore, sessionChecksPerLoop)
	if err != nil {
		slog.Error("Error getting cookies to verify", "error", err.Error())
		return
	}
	slog.Info("Checking user sessions", "count", len(users))
	for i, user := range users {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}
		bot.checkSession(user)
	}
}

// checkSession confirms the user's session is still logged in to SHiFT, and alerts them if it isn't
func (bot *Bot) checkSession(user store.UserCookies) {
	defer recoverUserPanic(user.UserID)

	platform, dm, err := bot.storage.GetUserPlatformAndDM(user.UserID)
	if err != nil {
		slog.Error("Error getting platform", "user_id", user.UserID, "error", err.Error())
		return
	}
	client, err := bot.newShiftClient(user.Cookies)
	if err != nil {
		slog.Error("Error creating shift client", "user_id", user.UserID, "error", err.Error())
		return
	}
	_, err = client.CheckRewards(shift.Platform(platform), shift.Borderlands4, 1)
	if errors.Is(err, shift.ErrNotLoggedIn) {
		slog.Info("User session is no longer logged in", "user_id", user.UserID)
		bot.alertUser(user.UserID, dm, store.AlertSessionExpired, sessionExpiredMessage)
		return
	}
	if err != nil {
		// probably temporary (SHiFT is down, etc), so leave it unverified to try again next loop
		slog.Warn("Error checking user session", "user_id", user.UserID, "error", err.Error())
		return
	}

	err = bot.storage.SetUserCookiesVerified(user.UserID)
	if err != nil {
		slog.Error("Error setting cookies verified", "user_id", user.UserID, "error", err.Error())
	}
	// the session works, so any sequential errors weren't from it expiring. Clear them so redemption can resume
	err = bot.storage.ClearShiftErrors(user.UserID)
	if err != nil {
		slog.Error("Error clearing shift errors from db", "user_id", user.UserID, "error", err.Error())
	}
}

// shiftClient is the subset of *shift.Client used to redeem codes
type shiftClient interface {
	CheckRewards(platform shift.Platform, game shift.Game, limit int) ([]shift.Reward, error)
	RedeemCode(code string, platform shift.Platform) (string, error)
}

// redeemCode redeems a code for a user, and attempts to determine what "reward" was indicated by the redemption
func (bot *Bot) redeemCode(client shiftClient, user store.UserCookies, code string, platform shift.Platform) (reward *shift.Reward, status string, err error) {
	rewards, err := client.CheckRewards(platform, shift.Borderlands4, -1)
	if err != nil {
		return nil, "", err
	}

	status, err = client.RedeemCode(code, platform)
	if err != nil {
		newRewards, err2 := client.CheckRewards(platform, shift.Borderlands4, -1)
		if err2 != nil {
			return nil, status, err2
		}

		// NOTE: if code redemption starts being performed in parallel for a given user, then this would be unsafe...

		if len(newRewards) > len(rewards) {
			slog.Info("Code redemption returned error, but reward length increased, so presumably it was successful", "error", err.Error())
			status = shift.SUCCESS
			reward = &newRewards[0]
			// fallthrough to below, where we add the redemption
		} else {
			return nil, status, err
		}
	}

	// only final outcomes are recorded as redemptions, because a code is never tried again for a user once recorded.
	// Anything else is returned as an error, so the code is retried later
	switch shift.DetermineResponseType(status) {
	case shift.Link2KAccount:
		return nil, status, errLink2KAccount
	case shift.Unrecognized:
		return nil, status, fmt.Errorf("unrecognized code redemption response: %q", status)
	}

	// only check the reward if we successfully redeemed (and don't already know it from above)
	if status == shift.SUCCESS && reward == nil {
		// the code was redeemed regardless, so a failed or empty rewards lookup only means we can't report the reward.
		// Don't return an error; that would skip recording the redemption, and count as a failure towards the user's errors
		newRewards, err2 := client.CheckRewards(platform, shift.Borderlands4, 1)
		if err2 != nil {
			slog.Warn("Code redemption succeeded, but failed to check rewards", "user_id", user.UserID, "code", code, "platform", platform, "error", err2.Error())
		} else if len(newRewards) > 0 {
			reward = &newRewards[0]
		} else {
			slog.Warn("Code redemption succeeded, but no rewards were listed", "user_id", user.UserID, "code", code, "platform", platform)
		}
	}

	err = bot.storage.AddRedemption(user.UserID, code, string(platform), status)
	if err != nil {
		slog.Error("Error adding redemption", "user_id", user.UserID, "code", code, "platform", platform, "status", status, "error", err.Error())
	} else {
		slog.Info("processed code", "user_id", user.UserID, "code", code, "status", status)
	}
	return reward, status, nil
}
