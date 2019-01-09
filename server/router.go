package app

import (
	"net/http"
	"time"

	"github.com/VojtechVitek/ratelimit"
	"github.com/VojtechVitek/ratelimit/memory"
	"github.com/gorilla/mux"
	"github.com/gosu-team/fptu-api/redis"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/gosu-team/fptu-api/controllers"
	"github.com/gosu-team/fptu-api/lib"
	"github.com/gosu-team/fptu-api/middlewares"
)

func notFound(w http.ResponseWriter, r *http.Request) {
	res := lib.Response{ResponseWriter: w}
	res.SendNotFound()
}

func privateRoute(controller http.HandlerFunc) http.Handler {
	return middlewares.JWTMiddleware().Handler(http.HandlerFunc(controller))
}

var pool = &redigo.Pool{
	MaxIdle:     50,
	MaxActive:   250,
	IdleTimeout: 300 * time.Second,
	Wait:        false, // Important
	Dial: func() (redigo.Conn, error) {
		c, err := redigo.DialTimeout("tcp", "127.0.0.1:6379", 200*time.Millisecond, 100*time.Millisecond, 100*time.Millisecond)
		if err != nil {
			return nil, err
		}
		return c, err
	},
	TestOnBorrow: func(c redigo.Conn, t time.Time) error {
		_, err := c.Do("PING")
		return err
	},
}

// NewRouter ...
func NewRouter() *mux.Router {

	// Create main router
	mainRouter := mux.NewRouter().StrictSlash(true)
	mainRouter.KeepContext = true

	mainRouter.Use(ratelimit.Request(ratelimit.IP).Rate(5, 5*time.Second).LimitBy(redis.New(pool), memory.New()))

	// Handle 404
	mainRouter.NotFoundHandler = http.HandlerFunc(notFound)

	/**
	 * meta-data
	 */
	mainRouter.Methods("GET").Path("/api/info").HandlerFunc(controllers.GetAPIInfo)

	/**
	 * /users
	 */
	// usersRouter.HandleFunc("/", l.Use(c.GetAllUsersHandler, m.SaySomething())).Methods("GET")

	// API Version
	apiPath := "/api"
	apiVersion := "/v1"
	apiPrefix := apiPath + apiVersion

	// Auth routes
	mainRouter.Methods("POST").Path("/auth/login").HandlerFunc(controllers.LoginHandler)
	mainRouter.Methods("POST").Path("/auth/login_facebook").HandlerFunc(controllers.LoginHandlerWithoutPassword)

	// User routes
	mainRouter.Methods("GET").Path(apiPrefix + "/users").Handler(privateRoute(controllers.GetAllUsersHandler))
	// mainRouter.Methods("POST").Path(apiPrefix + "/users").Handler(privateRoute(controllers.CreateUserHandler))
	// mainRouter.Methods("POST").Path(apiPrefix + "/users").HandlerFunc(controllers.CreateUserHandler)
	mainRouter.Methods("GET").Path(apiPrefix + "/users/{id}").Handler(privateRoute(controllers.GetUserByIDHandler))
	mainRouter.Methods("PUT").Path(apiPrefix + "/users/{id}").Handler(privateRoute(controllers.UpdateUserHandler))
	mainRouter.Methods("DELETE").Path(apiPrefix + "/users/{id}").Handler(privateRoute(controllers.DeleteUserHandler))

	// Confession routes
	mainRouter.Methods("GET").Path(apiPrefix + "/admincp/confessions").Handler(privateRoute(controllers.GetAllConfessionsHandler))
	mainRouter.Methods("POST").Path(apiPrefix + "/confessions").HandlerFunc(controllers.CreateConfessionHandler)
	mainRouter.Methods("POST").Path(apiPrefix + "/myconfess").HandlerFunc(controllers.GetConfessionsBySenderHandler)
	mainRouter.Methods("GET").Path(apiPrefix + "/confessions/overview").HandlerFunc(controllers.GetConfessionsOverviewHandler)
	mainRouter.Methods("PUT").Path(apiPrefix + "/admincp/confessions/approve").Handler(privateRoute(controllers.ApproveConfessionHandler))
	mainRouter.Methods("PUT").Path(apiPrefix + "/admincp/confessions/rollback_approve").Handler(privateRoute(controllers.RollbackApproveConfessionHandler))
	mainRouter.Methods("PUT").Path(apiPrefix + "/admincp/confessions/reject").Handler(privateRoute(controllers.RejectConfessionHandler))

	// Get NextID
	mainRouter.Methods("GET").Path(apiPrefix + "/next_confession_id").HandlerFunc(controllers.GetNextConfessionNextIDHandler)

	// Crawl
	mainRouter.Methods("GET").Path("/crawl").HandlerFunc(controllers.GetPostsByURLHandler)

	return mainRouter
}
