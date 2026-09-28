package store

import (
	"crypto/rand"
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/denverquane/slickshift/shift"
)

func newTestDB(t *testing.T) Store {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatal(err)
	}
	encryptor, err := NewEncryptor(key)
	if err != nil {
		log.Fatal(err)
	}
	store, err := NewSqliteStore(":memory:", encryptor)
	if err != nil {
		log.Fatal(err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store
}

func TestSqliteStore_AddUser(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"

	if st.UserExists(userID) {
		t.Fatal("User exists when db is fresh")
	}

	err := st.AddUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.UserExists(userID) {
		t.Fatal("User does not exist after added to DB")
	}
}

func TestSqliteStore_SetUserPlatform(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	const platform = string(shift.Steam)

	st.AddUser(userID)
	p, _, err := st.GetUserPlatformAndDM(userID)
	if err != nil {
		t.Fatal(err)
	}
	if p != "" {
		t.Fatal("User platform should be empty")
	}

	err = st.SetUserPlatform(userID, platform)
	if err != nil {
		t.Fatal(err)
	}
	p, _, err = st.GetUserPlatformAndDM(userID)
	if err != nil {
		t.Fatal(err)
	}
	if p != platform {
		t.Fatal("User platform should be " + platform)
	}
}

func TestSqliteStore_GetUserPlatformAndDM(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	const platform = string(shift.Steam)

	st.AddUser(userID)

	p, dm, err := st.GetUserPlatformAndDM(userID)
	if err != nil {
		t.Fatal(err)
	}
	if p != "" {
		t.Fatal("User platform should be empty")
	}
	if dm {
		t.Fatal("DM should be false")
	}
	st.SetUserPlatform(userID, platform)

	p, dm, err = st.GetUserPlatformAndDM(userID)
	if err != nil {
		t.Fatal(err)
	}
	if p != platform {
		t.Fatal("User platform should be " + platform)
	}
	if dm {
		t.Fatal("DM should be false")
	}

	st.SetUserDM(userID, true)
	p, dm, err = st.GetUserPlatformAndDM(userID)
	if err != nil {
		t.Fatal(err)
	}
	if p != platform {
		t.Fatal("User platform should be " + platform)
	}
	if !dm {
		t.Fatal("DM should be true")
	}

}

func TestSqliteStore_AddCode(t *testing.T) {
	st := newTestDB(t)
	const code = "AAAAA"
	const game = string(shift.Borderlands4)

	if st.CodeExists(code) {
		t.Fatal("Code exists when db is fresh")
	}

	err := st.AddCode(code, game, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !st.CodeExists(code) {
		t.Fatal("Code does not exist after added to DB")
	}
}

func TestNewSqliteStore_AddRedemptionAndGetSuccessStatus(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	const code = "AAAAA"
	const code2 = "BBBBB"
	const game = string(shift.Borderlands4)
	const platform = string(shift.Steam)
	const status = shift.SUCCESS
	const otherStatus = shift.ALREADY_REDEEMED

	st.AddUser(userID)
	st.AddCode(code, game, nil, nil)
	st.AddCode(code2, game, nil, nil)

	err := st.AddRedemption(userID, code, platform, status)
	if err != nil {
		t.Fatal(err)
	}
	err = st.AddRedemption(userID, code2, platform, otherStatus)
	if err != nil {
		t.Fatal(err)
	}

	redemptions, err := st.GetRecentRedemptionsForUser(userID, status, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(redemptions) != 1 {
		t.Fatal("Redemptions should contain 1 item")
	}
	r := redemptions[0]
	if r.Status != status {
		t.Fatal("Redemption status should be " + status)
	}
	if r.Code != code {
		t.Fatal("Redemption code should be " + code)
	}
	if r.Game != game {
		t.Fatal("Redemption game should be " + game)
	}
	if r.Platform != platform {
		t.Fatal("Redemption platform should be " + platform)
	}
	redemptions, err = st.GetRecentRedemptionsForUser(userID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(redemptions) != 2 {
		t.Fatal("Redemptions should contain 2 items when unfiltered")
	}

}

// when fetching codes for a user to redeem, if other users have marked the codes as expired or invalid, those
// codes should not be retrieved
func TestSqliteStore_GetValidCodes(t *testing.T) {
	// arrange
	st := newTestDB(t)
	const userID = "123"
	const testUserID = "234"

	const goodCode = "ABCDEF"
	const expiredCode = "BBBBB"
	const notExistCode = "CCCCC"

	const platform = string(shift.Steam)
	const game = string(shift.Borderlands4)

	st.AddUser(userID)
	st.AddUser(testUserID)

	st.SetUserPlatform(userID, platform)
	st.SetUserPlatform(testUserID, platform)

	st.AddCode(goodCode, game, nil, nil)
	st.AddCode(expiredCode, game, nil, nil)
	st.AddCode(notExistCode, game, nil, nil)

	st.AddRedemption(userID, goodCode, platform, shift.SUCCESS)
	st.AddRedemption(userID, expiredCode, platform, shift.EXPIRED)
	st.AddRedemption(userID, notExistCode, platform, shift.NOT_EXIST)

	// act
	codes, err := st.GetValidCodesNotRedeemedForUser(testUserID, platform, 10)
	if err != nil {
		t.Fatal(err)
	}

	// assert
	if len(codes) != 1 {
		t.Fatal("Expected 1 code, got ", len(codes))
	}
	if codes[0] != goodCode {
		t.Fatal("Expected good code, got ", codes[0])
	}
}

func TestSqliteStore_AddError(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	const code = "XXXXX"
	const game = string(shift.Borderlands4)
	const platform = string(shift.Steam)
	const errText = "234"

	st.AddUser(userID)
	st.AddCode(code, game, nil, nil)

	err := st.AddShiftError(userID, code, platform, errText)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSqlite_GetShiftError(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	const code = "XXXXX"
	const game = string(shift.Borderlands4)
	const platform = string(shift.Steam)
	const errText = "234"

	st.AddUser(userID)
	st.AddCode(code, game, nil, nil)
	st.AddShiftError(userID, code, platform, errText)

	errs, err := st.GetShiftErrors(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 1 {
		t.Fatal("Expected 1 code, got ", len(errs))
	}
	if errs[0] != errText {
		t.Fatal("Expected err text, got ", errs[0])
	}
}

func TestSqliteStore_ClearShiftErrors(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	const code = "XXXXX"
	const game = string(shift.Borderlands4)
	const platform = string(shift.Steam)
	const errText = "234"

	st.AddUser(userID)
	st.AddCode(code, game, nil, nil)
	st.AddShiftError(userID, code, platform, errText)

	err := st.ClearShiftErrors(userID)
	if err != nil {
		t.Fatal(err)
	}
	errs, err := st.GetShiftErrors(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 0 {
		t.Fatal("Expected 0 code, got ", len(errs))
	}
}

// Regression: the success and already redeemed counts were swapped
func TestSqliteStore_RedemptionSummaryForUser(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	codes := []string{"AAAAA", "BBBBB", "CCCCC"}

	st.AddUser(userID)
	for _, code := range codes {
		st.AddCode(code, string(shift.Borderlands4), nil, nil)
	}
	st.AddRedemption(userID, codes[0], string(shift.Steam), shift.SUCCESS)
	st.AddRedemption(userID, codes[1], string(shift.Steam), shift.ALREADY_REDEEMED)
	st.AddRedemption(userID, codes[2], string(shift.Steam), shift.ALREADY_REDEEMED)

	summary, err := st.RedemptionSummaryForUser(userID)
	if err != nil {
		t.Fatal(err)
	}
	if summary["total"] != 3 {
		t.Fatal("Expected 3 total, got ", summary["total"])
	}
	if summary["success"] != 1 {
		t.Fatal("Expected 1 success, got ", summary["success"])
	}
	if summary["already_redeemed"] != 2 {
		t.Fatal("Expected 2 already redeemed, got ", summary["already_redeemed"])
	}
}

func TestSqliteStore_SetUserAlert(t *testing.T) {
	st := newTestDB(t)
	const userID = "123"
	st.AddUser(userID)

	alert, err := st.GetUserAlert(userID)
	if err != nil {
		t.Fatal(err)
	}
	if alert != "" {
		t.Fatal("Alert should be empty for new user, got " + alert)
	}

	changed, err := st.SetUserAlert(userID, AlertSessionExpired)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("Setting a new alert should report it changed")
	}
	alert, _ = st.GetUserAlert(userID)
	if alert != AlertSessionExpired {
		t.Fatal("Alert should be " + AlertSessionExpired + ", got " + alert)
	}

	// this is what prevents notifying the user more than once
	changed, _ = st.SetUserAlert(userID, AlertSessionExpired)
	if changed {
		t.Fatal("Setting the same alert again should not report it changed")
	}

	changed, _ = st.SetUserAlert(userID, AlertLink2KAccount)
	if !changed {
		t.Fatal("Setting a different alert should report it changed")
	}

	err = st.ClearUserAlert(userID)
	if err != nil {
		t.Fatal(err)
	}
	alert, _ = st.GetUserAlert(userID)
	if alert != "" {
		t.Fatal("Alert should be empty after clearing, got " + alert)
	}
	changed, _ = st.SetUserAlert(userID, AlertLink2KAccount)
	if !changed {
		t.Fatal("Setting an alert again after clearing should report it changed")
	}
}

func TestSqliteStore_GetDecryptedUserCookiesToVerify(t *testing.T) {
	st := newTestDB(t)
	cookies := []*http.Cookie{{Name: "si", Value: "si_here"}, {Name: "_session_id", Value: "session_id_here"}}
	const verified, unverified, noPlatform, expired, link2K = "1", "2", "3", "4", "5"
	for _, userID := range []string{verified, unverified, noPlatform, expired, link2K} {
		st.AddUser(userID)
		if userID != noPlatform {
			st.SetUserPlatform(userID, string(shift.Steam))
		}
		err := st.EncryptAndSetUserCookies(userID, cookies)
		if err != nil {
			t.Fatal(err)
		}
	}
	st.SetUserAlert(expired, AlertSessionExpired)
	st.SetUserAlert(link2K, AlertLink2KAccount)
	// like cookies that were set before verified_unix existed
	_, err := st.(*Sqlite).db.Exec("UPDATE user_cookies SET verified_unix = NULL WHERE user_id IN (?, ?, ?, ?)", unverified, noPlatform, expired, link2K)
	if err != nil {
		t.Fatal(err)
	}

	users, err := st.GetDecryptedUserCookiesToVerify(time.Now().Add(-time.Hour).Unix(), 10)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, user := range users {
		ids[user.UserID] = true
	}
	if len(users) != 2 || !ids[unverified] || !ids[link2K] {
		t.Fatalf("Expected only unverified users with a platform and without expired sessions (%s, %s), got %+v", unverified, link2K, users)
	}
	if len(users[0].Cookies) != 2 || users[0].Cookies[0].Value != "si_here" {
		t.Fatalf("Cookies should be decrypted, got %+v", users[0].Cookies)
	}

	err = st.SetUserCookiesVerified(unverified)
	if err != nil {
		t.Fatal(err)
	}
	users, _ = st.GetDecryptedUserCookiesToVerify(time.Now().Add(-time.Hour).Unix(), 10)
	if len(users) != 1 || users[0].UserID != link2K {
		t.Fatalf("Expected only %s after verifying %s, got %+v", link2K, unverified, users)
	}

	// everyone with a platform (and without an expired session) is due if the verification cutoff is in the future
	users, _ = st.GetDecryptedUserCookiesToVerify(time.Now().Add(time.Hour).Unix(), 10)
	if len(users) != 3 {
		t.Fatalf("Expected 3 users, got %+v", users)
	}
	users, _ = st.GetDecryptedUserCookiesToVerify(time.Now().Add(time.Hour).Unix(), 1)
	if len(users) != 1 || users[0].UserID != link2K {
		t.Fatalf("Expected least recently verified user (%s) first, got %+v", link2K, users)
	}
}

// Migration 3 removes redemptions that weren't final outcomes, so those codes get retried
func TestSqliteStore_Migration3RemovesNonFinalRedemptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range []string{"sqlite/1.sql", "sqlite/2.sql"} {
		contents, err := schemaFS.ReadFile(migration)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(string(contents))
		if err != nil {
			t.Fatal(err)
		}
	}
	setVersion(db, 2)
	statuses := map[string]string{
		"AAAAA": shift.SUCCESS,
		"BBBBB": shift.ALREADY_REDEEMED,
		"CCCCC": shift.EXPIRED,
		"DDDDD": shift.NOT_EXIST,
		"EEEEE": shift.LINK2K,
		"FFFFF": "",
		"GGGGG": "Some unrecognized response",
	}
	db.Exec("INSERT INTO users (id, updated_unix, created_unix) VALUES (1, 0, 0)")
	for code, status := range statuses {
		db.Exec("INSERT INTO shift_codes (code, game, created_unix) VALUES (?, ?, 0)", code, shift.Borderlands4)
		_, err = db.Exec("INSERT INTO redemptions (code, user_id, platform, status, created_unix) VALUES (?, 1, ?, ?, 0)", code, shift.Steam, status)
		if err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	key := make([]byte, 32)
	rand.Read(key)
	encryptor, _ := NewEncryptor(key)
	st, err := NewSqliteStore(path, encryptor)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	redemptions, err := st.GetRecentRedemptionsForUser("1", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	remaining := map[string]string{}
	for _, r := range redemptions {
		remaining[r.Code] = r.Status
	}
	for _, code := range []string{"AAAAA", "BBBBB", "CCCCC", "DDDDD"} {
		if remaining[code] != statuses[code] {
			t.Errorf("Final redemption for %s (%q) should be kept", code, statuses[code])
		}
	}
	if len(remaining) != 4 {
		t.Errorf("Only the 4 final redemptions should be kept, got %+v", remaining)
	}
}
