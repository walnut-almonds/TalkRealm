package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/walnut-almonds/talkrealm/internal/handler"
	"github.com/walnut-almonds/talkrealm/internal/service"
)

type mockLearnService struct {
	createLevelFn     func(userID uint, mode string, tier int, locale string) (*service.LevelView, error)
	createCrosswordFn func(userID uint, tier int, locale string) (*service.CrosswordView, error)
	guessFn           func(userID uint, levelID string, req *service.LearnGuessRequest) (*service.GuessOutcome, error)
	statsFn           func(userID uint) (*service.LearnStatsView, error)
}

func (m *mockLearnService) CreateLevel(
	userID uint,
	mode string,
	tier int,
	locale string,
) (*service.LevelView, error) {
	return m.createLevelFn(userID, mode, tier, locale)
}

func (m *mockLearnService) CreateCrosswordLevel(
	userID uint,
	tier int,
	locale string,
) (*service.CrosswordView, error) {
	return m.createCrosswordFn(userID, tier, locale)
}

func (m *mockLearnService) Guess(
	userID uint,
	levelID string,
	req *service.LearnGuessRequest,
) (*service.GuessOutcome, error) {
	return m.guessFn(userID, levelID, req)
}

func (m *mockLearnService) Stats(userID uint) (*service.LearnStatsView, error) {
	return m.statsFn(userID)
}

func (m *mockLearnService) DailyLevel(userID uint, locale string) (*service.DailyView, error) {
	return nil, nil //nolint:nilnil // 測試 stub，尚無 daily handler 測試
}

func (m *mockLearnService) DailyLeaderboard(userID uint) (*service.LeaderboardView, error) {
	return nil, nil //nolint:nilnil // 測試 stub，尚無 daily handler 測試
}

func setupLearnRouter(svc service.LearnService) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	auth := authMiddleware(uint(7))
	h := handler.NewLearnHandler(svc)
	r.POST("/learn/levels", auth, h.CreateLevel)
	r.POST("/learn/levels/crossword", auth, h.CreateCrossword)
	r.POST("/learn/levels/:id/guess", auth, h.Guess)
	r.GET("/learn/stats", auth, h.GetStats)

	return r
}

func TestCreateLevelOK(t *testing.T) {
	svc := &mockLearnService{
		createLevelFn: func(userID uint, mode string, tier int, locale string) (*service.LevelView, error) {
			if userID != 7 || mode != service.ModeFill || tier != 2 {
				t.Errorf("args: %d %s %d", userID, mode, tier)
			}

			return &service.LevelView{LevelID: "abc", Mode: mode, Tier: tier}, nil
		},
	}

	body, _ := json.Marshal(map[string]any{"mode": service.ModeFill, "tier": 2})
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/learn/levels",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	setupLearnRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestCreateLevelInvalidMode(t *testing.T) {
	svc := &mockLearnService{
		createLevelFn: func(uint, string, int, string) (*service.LevelView, error) {
			return nil, service.ErrLearnInvalidMode
		},
	}

	body, _ := json.Marshal(map[string]any{"mode": "bogus", "tier": 2})
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/learn/levels",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	setupLearnRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestGuessExpiredLevel(t *testing.T) {
	svc := &mockLearnService{
		guessFn: func(uint, string, *service.LearnGuessRequest) (*service.GuessOutcome, error) {
			return nil, service.ErrLearnLevelNotFound
		},
	}

	body, _ := json.Marshal(map[string]any{"slot": 0, "word": "star"})
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/learn/levels/xyz/guess",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	setupLearnRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusGone { // 410（spec §5）
		t.Errorf("status = %d", w.Code)
	}
}

func TestCreateCrosswordOK(t *testing.T) {
	svc := &mockLearnService{
		createCrosswordFn: func(userID uint, tier int, locale string) (*service.CrosswordView, error) {
			if userID != 7 || tier != 2 {
				t.Errorf("args: %d %d", userID, tier)
			}

			return &service.CrosswordView{LevelID: "cw1", Tier: tier}, nil
		},
	}

	body, _ := json.Marshal(map[string]any{"tier": 2})
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/learn/levels/crossword",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	setupLearnRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestCreateCrosswordInvalidTier(t *testing.T) {
	svc := &mockLearnService{
		createCrosswordFn: func(uint, int, string) (*service.CrosswordView, error) {
			return nil, service.ErrLearnInvalidTier
		},
	}

	body, _ := json.Marshal(map[string]any{"tier": 9})
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/learn/levels/crossword",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	setupLearnRouter(svc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}
