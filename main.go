package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/gorp.v1"

	"github.com/dgrijalva/jwt-go"

	"strconv"

	_ "github.com/lib/pq"
	"time"
	"strings"
	"fmt"
)

// ========================= INITIALIZE

/* Set up a global string for our secret */
var mySigningKey = []byte("APIRI4008090121721000STDGTL")

var db = initDb()
var dbmap = initDbmap()

func initDb() *sql.DB {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))

	checkErr(err, "Failed open db")
	return db
}

func initDbmap() *gorp.DbMap {
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}

	dbmap.AddTableWithName(ProviderAccount{}, "provideraccount").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(ProviderData{}, "providerdata").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(ProviderLocation{}, "providerlocation").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(KategoriJasa{}, "kategorijasa").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(ProviderPriceList{}, "providerpricelist").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(ProviderRating{}, "providerrating").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(ProviderGallery{}, "providergallery").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(ProviderProfileImage{}, "providerprofileimage").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(OrderVendor{}, "ordervendor").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(OrderVendorDetail{}, "ordervendordetail").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(OrderVendorJourney{}, "ordervendorjourney").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(OrderVendorTracking{}, "ordervendortracking").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(UserAccount{}, "useraccount").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(UserProfile{}, "userprofile").SetKeys(true, "UserId")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(AuthToken{}, "authtoken").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	dbmap.AddTableWithName(AuthTokenProvider{}, "authtokenprovider").SetKeys(true, "Id")
	checkErr(dbmap.CreateTablesIfNotExists(), "Create tables failed")

	return dbmap
}

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func main() {
	r := gin.Default()
	v1 := r.Group("api/v1")
	{
		v1.POST("/user/signin/email", PostSignInEmail)
		v1.POST("/user/signup/email", PostSignUpEmail)
		v1.POST("/user/auth/social", PostAuthSocial)
		v1.POST("/provider/create", PostCreateProvider)
		v1.POST("/provider/signin", PostSignInProvider)
		v1.PUT("/provider/inactive", InActiveProvider)
		v1.PUT("/provider/active", ActiveProvider)
		v1.POST("/jasa/create", PostCreateNewJasa)
	}
	v1.Use(TokenAuthUserMiddleware())
	{
		v1.GET("/providers", GetProviders)
		v1.GET("/providers/near", GetNearProviderForMap)
		v1.GET("/providers/search", GetProvidersByKeyword)
		v1.GET("/provider/jasa/:jasa_id", GetProvidersByCategory)
		v1.GET("/provider/data/:id", GetProvider)
		v1.GET("/provider/prices/:provider_id", GetProviderPriceList)
		v1.POST("/provider/rating/add", PostAddedRating)
		v1.GET("/rating/get/:provider_id", GetProviderRating)
		v1.PUT("/provider/rating/edit", UpdateProviderRating)
		v1.GET("/gallery/data/:provider_id", GetListImageGallery)
		v1.GET("/profile/data/:provider_id", GetProfileProvider)
		v1.POST("/order/new", PostNewOrder)
		v1.PUT("/user/profile/update", PutProfileUpdate)
		v1.PUT("/user/devicetoken/update", PutDeviceTokenUpdate)
		v1.GET("/user/me", GetUserProfile)
	}
	v1.Use(TokenAuthProviderMiddleware())
	{
		v1.POST("/provider/mylocation", PostMyLocationProvider)
		v1.POST("/provider/price/add", PostAddProviderPriceList)
		v1.GET("/price/data/:provider_id/:id", GetProviderPrice)
		v1.PUT("/provider/price/edit", UpdateProviderPrice)
		v1.POST("/provider/gallery/add", PostProviderImageGallery)
		v1.DELETE("/gallery/delete", DeleteImageGallery)
		v1.POST("/provider/profile/add", PostProfileProvider)
		v1.PUT("/provider/edit/:provider_id", UpdateProviderData)
		v1.POST("/order/status", PostNewOrderJourney)
		v1.PUT("/order/tracking", UpdateOrderTracking)
	}

	r.Run(GetPort())

}

func getTokenFromHeader(c *gin.Context) string {
	var tokenStr string
	bearer := c.Request.Header.Get("Authorization")

	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		tokenStr = bearer[7:]
	}
	return tokenStr
}

func TokenAuthProviderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := getTokenFromHeader(c)

		if tokenStr == "" {
			c.JSON(401, gin.H{"error" : "Unauthorize request. Please check your header request, and make sure include Authorization token in your request."})
			c.Abort()
			return
		} else {
			var authTokenProvider AuthTokenProvider
			err := dbmap.SelectOne(&authTokenProvider, `SELECT id, provider_id, expired_date FROM authtokenprovider
				WHERE auth_token=$1`, tokenStr)

			if err != nil {
				c.JSON(401, gin.H{"error" : "Unauthorize request. Please check your header request, and make sure include Authorization token in your request."})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func TokenAuthUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		tokenStr := getTokenFromHeader(c)

		if tokenStr == "" {
			c.JSON(400, gin.H{"error" : "Unauthorize request. Please check your header request, and make sure include Authorization token in your request."})
			c.Abort()
			return
		} else {
			var authToken AuthToken
			err := dbmap.SelectOne(&authToken, `SELECT id, user_id, expired_date FROM authtoken
				WHERE auth_token=$1`, tokenStr)

			if err != nil {
				c.JSON(401, gin.H{"error" : "Unauthorize request. Please check your header request, and make sure include Authorization token in your request."})
				c.Abort()
				return
			} else {

				if time.Now().Unix() >= authToken.ExpireDate {
					removeExpiredToken(authToken.Id);
					c.JSON(401, gin.H{"error" : "Expired API token"})
					c.Abort()
					return;
				}
			}
		}

		c.Next()
	}
}

func removeExpiredToken(tokenId int64) {
	db.QueryRow(`DELETE FROM authtoken WHERE id=$1`, tokenId);
}

func GetPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "4747"
	}
	return ":" + port
}

// =============================== STRUCT

/**
Provider account
Id
ProviderId
Email
Token
DeviceId
Status
 */
type ProviderAccount struct {
	Id          int64  `db:"id" json:"id"`
	ProviderId  int64  `db:"provider_id" json:"provider_id"`
	Email       string `db:"email" json:"email"`
	Password    string `db:"password" json:"password"`
	DeviceToken string `db:"device_token" json:"device_token"`
	Status      int64   `db:"status" json:"status"`
}

/**
Provider data
Id
Nama
Email
PhoneNumber
JasaId
Alamat
Provinsi
Kabupaten
KodePos
Dokumen
JoinDate
ModifiedDate
 */
type ProviderData struct {
	Id           int64  `db:"id" json:"id"`
	Nama         string `db:"nama" json:"nama"`
	Email        string `db:"email" json:"email"`
	PhoneNumber  string `db:"phone_number" json:"phone_number"`
	JasaId       int64   `db:"jasa_id" json:"jasa_id"`
	Alamat       string `db:"alamat" json:"alamat"`
	Provinsi     string `db:"provinsi" json:"provinsi"`
	Kabupaten    string `db:"kabupaten" json:"kabupaten"`
	Kelurahan    string `db:"kelurahan" json:"kelurahan"`
	KodePos      string `db:"kode_pos" json:"kode_pos"`
	Dokumen      string `db:"dokumen" json:"dokumen"`
	JoinDate     int64  `db:"join_date" json:"join_date"`
	ModifiedDate int64  `db:"modified_date" json:"modified_date"`
}

/**
Provider location
Id
ProviderId
Latitude
Longitude
 */
type ProviderLocation struct {
	Id         int64   `db:"id" json:"id"`
	ProviderId int64   `db:"provider_id" json:"provider_id"`
	Latitude   float64 `db:"latitude" json:"latitude"`
	Longitude  float64 `db:"longitude" json:"longitude"`
}

/**
ProviderLatLang
Latitude
Longitude
 */
type ProviderLatLng struct {
	Latitude  float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
}

/**
Kategori Jasa
Id
Jenis
 */
type KategoriJasa struct {
	Id    int64  `db:"id" json:"id"`
	Jenis string `db:"jenis" json:"jenis"`
}

/**
UserLocation
UserId
Latitude
Longitude
 */
type UserLocation struct {
	UserId    int64   `db:"user_id" json:"user_id"`
	Latitude  float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
}

/**
Provider near user for map
Id
Nama
JasaId
JenisJasa
Latitude
Longitude
Distance
 */
type NearProviderForMap struct {
	Id        int64   `db:"id" json:"id"`
	Nama      string  `db:"nama" json:"nama"`
	JasaId    int64   `db:"jasa_id" json:"jasa_id"`
	JenisJasa string  `db:"jenis_jasa" json:"jenis_jasa"`
	Latitude  float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
	Distance  float64 `db:"distance" json:"distance"`
}

/**
Provider near user by type
JasaId
JenisJasa
CountJasaProvider
MinDistance
 */
type NearProviderByType struct {
	JasaId            int64   `db:"jasa_id" json:"jasa_id"`
	JenisJasa         string  `db:"jenis_jasa" json:"jenis_jasa"`
	CountJasaProvider int64    `db:"count_jasa_provider" json:"count_jasa_provider"`
	MinDistance       float64 `db:"min_distance" json:"min_distance"`
}

/**
List price provider
Id
ProviderId
ServiceName
ServicePrice
Negotiable
 */
type ProviderPriceList struct {
	Id           int64  `db:"id" json:"id"`
	ProviderId   int64  `db:"provider_id" json:"provider_id"`
	ServiceName  string `db:"service_name" json:"service_name"`
	ServicePrice int64  `db:"service_price" json:"service_price"`
	Negotiable   int64   `db:"negotiable" json:"negotiable"`
}

/**
Rating provider
Id
ProviderId
UserId
UserRating
 */
type ProviderRating struct {
	Id         int64 `db:"id" json:"id"`
	ProviderId int64 `db:"provider_id" json:"provider_id"`
	UserId     int64 `db:"user_id" json:"user_id"`
	UserRating int64  `db:"user_rating" json:"user_rating"`
}

/**
Provider gallery
Id
ProviderId
Image
 */
type ProviderGallery struct {
	Id         int64  `db:"id" json:"id"`
	ProviderId int64  `db:"provider_id" json:"provider_id"`
	Image      string `db:"image" json:"image"`
}

/**
Provider profile image
Id
ProviderId
ProfilePict
ProfileBg
 */
type ProviderProfileImage struct {
	Id          int64  `db:"id" json:"id"`
	ProviderId  int64  `db:"provider_id" json:"provider_id"`
	ProfilePict string `db:"profile_pict" json:"profile_pict"`
	ProfileBg   string `db:"profile_bg" json:"profile_bg"`
}

/**
Provider Basic Info Response
Id
Nama
Alamat
JasaId
JenisJasa
 */
type ProviderBasicInfo struct {
	Id        int64  `db:"id" json:"id"`
	Nama      string `db:"nama" json:"nama"`
	Alamat    string `db:"alamat" json:"alamat"`
	JasaId    int64   `db:"jasa_id" json:"jasa_id"`
	JenisJasa string `db:"jenis_jasa" json:"jenis_jasa"`
}

/**
Order service
Id
ProviderId
UserId
Destination
DestinationLat
DestinationLong
DestinationDesc
Notes
PaymentMethod
OrderDate
 */
type OrderVendor struct {
	Id              int64    `db:"id" json:"id"`
	ProviderId      int64    `db:"provider_id" json:"provider_id"`
	UserId          int64    `db:"user_id" json:"user_id"`
	Destination     string  `db:"destination" json:"destination"`
	DestinationLat  float64 `db:"destination_lat" json:"destination_lat"`
	DestinationLong float64 `db:"destination_long" json:"destination_long"`
	DestinationDesc string  `db:"destination_desc" json:"destination_desc"`
	Notes           string  `db:"notes" json:"notes"`
	PaymentMethod   int     `db:"payment_method" json:"payment_method"`
	OrderDate       int64   `db:"order_date" json:"order_date"`
}

/**
Order detail
Id
OrderId
JasaId
ServiceName
ServicePrice
Qty
ModifiedDate
 */
type OrderVendorDetail struct {
	Id           int64   `db:"id" json:"id"`
	OrderId      int64   `db:"order_id" json:"order_id"`
	JasaId       int64   `db:"jasa_id" json:"jasa_id"`
	ServiceName  string `db:"service_name" json:"service_name"`
	ServicePrice int64  `db:"service_price" json:"service_price"`
	Qty          int64   `db:"qty" json:"qty"`
	ModifiedDate int64  `db:"modified_date" json:"modified_date"`
}

/**
Order vendor journey
Id
OrderId
Status
 */
type OrderVendorJourney struct {
	Id      int64 `db:"id" json:"id"`
	OrderId int64 `db:"order_id" json:"order_id"`
	Status  int64 `db:"status"`
}

/**
Order vendor tracking location
Id
OrderId
CurrentLatitude
CurrentLongitude
 */
type OrderVendorTracking struct {
	Id               int64    `db:"id" json:"id"`
	OrderId          int64    `db:"order_id" json:"order_id"`
	CurrentLatitude  float64 `db:"latitude" json:"latitude"`
	CurrentLongitude float64 `db:"longitude" json:"longitude"`
}

/**
Post transaction request
ProviderId
UserId
Destination
DestinationLat
DestinationLong
DestinationDesc
Notes
PaymentMethod
Notes
Data
OrderDate
 */
type PostTransaction struct {
	ProviderId      int64                  `json:"provider_id"`
	Destination     string                  `json:"destination"`
	DestinationLat  float64                 `json:"destination_lat"`
	DestinationLong float64                 `json:"destination_long"`
	DestinationDesc string                  `json:"destination_desc"`
	Notes           string                  `json:"notes"`
	PaymentMethod   int                     `json:"payment_method"`
	Data            []PostTransactionDetail `json:"data"`
	OrderDate       int64                   `json:"order_date"`
}

/**
Post transaction detail
JasaId
ServiceName
ServicePrice
Qty
ModifiedDate
 */
type PostTransactionDetail struct {
	JasaId       int64   `json:"jasa_id"`
	ServiceName  string `json:"service_name"`
	ServicePrice int64  `json:"service_price"`
	Qty          int64   `json:"qty"`
	ModifiedDate int64  `json:"modified_date"`
}

/**
Provider by type
Id
Nama
Latitude
Longitude
MinPrice
MaxPrice
Rating
Distance
 */
type ProviderByCat struct {
	Id        int64           `db:"id" json:"id"`
	Nama      string          `db:"nama" json:"nama"`
	Latitude  float64         `db:"latitude" json:"latitude"`
	Longitude float64         `db:"longitude" json:"longitude"`
	MinPrice  sql.NullInt64   `db:"min_price" json:"min_price"`
	MaxPrice  sql.NullInt64   `db:"max_price" json:"max_price"`
	Rating    sql.NullFloat64 `db:"rating" json:"rating"`
	Distance  float64         `db:"distance" json:"distance"`
}

/**
Provider by type Response
Id
Nama
Latitude
Longitude
MinPrice
MaxPrice
Rating
Distance
 */
type ListProviderByCat struct {
	Id        int64   `db:"id" json:"id"`
	Nama      string  `db:"nama" json:"nama"`
	Latitude  float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
	MinPrice  int64   `db:"min_price" json:"min_price"`
	MaxPrice  int64   `db:"max_price" json:"max_price"`
	Rating    float64 `db:"rating" json:"rating"`
	Distance  float64 `db:"distance" json:"distance"`
}

/**
User account
Id
Email
Password
AuthMode
DeviceToken
JoinDate
 */
type UserAccount struct {
	Id          int64   `db:"id" json:"id"`
	Email       string `db:"email" json:"email"`
	Password    string `db:"password" json:"password"`
	AuthMode    string `db:"auth_mode" json:"auth_mode"`
	DeviceToken string `db:"device_token" json:"device_token"`
	JoinDate    int64  `db:"join_date" json:"join_date"`
}

/**
Auth token
Id
UserId
AuthToken
ExpireDate
 */
type AuthToken struct {
	Id         int64   `db:"id" json:"id"`
	UserId     int64   `db:"user_id" json:"user_id"`
	AuthToken  string `db:"auth_token" json:"auth_token"`
	ExpireDate int64  `db:"expired_date" json:"expired_date"`
}

/**
Auth token for provider
Id
ProviderId
AuthToken
ExpireDate
 */
type AuthTokenProvider struct {
	Id         int64   `db:"id" json:"id"`
	ProviderId int64   `db:"provider_id" json:"provider_id"`
	AuthToken  string `db:"auth_token" json:"auth_token"`
	ExpireDate int64  `db:"expired_date" json:"expired_date"`
}

/**
Auth token response
AuthToken
ExpiredDate
 */
type AuthTokenRes struct {
	Token       string `json:"token"`
	ExpiredDate int64 `json:"expired_date"`
}

/**
User profile
UserId
FullName
Address
Latitude
Longitude
DOB
PhoneNumber
 */
type UserProfile struct {
	UserId      int64    `db:"user_id" json:"user_id"`
	FullName    string  `db:"full_name" json:"full_name"`
	Address     string  `db:"address" json:"address"`
	DOB         string  `db:"dob" json:"dob"`
	PhoneNumber string  `db:"phone_number" json:"phone_number"`
}

/**
Login Account Response
UserId
FullName
Email
PhoneNumber
AuthMode
AuthToken
DeviceToken
 */
type LoginAccount struct {
	UserId      int64      `json:"id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	AuthMode    string    `json:"auth_mode"`
	AuthToken   AuthTokenRes `json:"auth_token"`
}

/**
Provider Login Account
ProviderId
FullName
JasaId
JasaName
Email
PhoneNumber
AuthToken
 */
type ProviderLoginAccount struct {
	ProviderId  int64      `json:"id"`
	FullName    string    `json:"full_name"`
	JasaId      int64      `json:"jasa_id"`
	JasaName    string    `json:"jasa_nama"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	AuthToken   AuthTokenRes `json:"auth_token"`
}

// ========================== FUNC

func GetProviders(c *gin.Context) {
	// Get all list providers
}

func GetProvider(c *gin.Context) {

	// Get provider by id
	// providerId := c.Params.ByName("id")

	providerId := 23

	// Get basic info
	var providerBasicInfo ProviderBasicInfo
	errBasicInfo := dbmap.SelectOne(&providerBasicInfo,
		`SELECT pd.id as id, pd.nama, pd.alamat, pd.jasa_id, kj.jenis as jenis_jasa
		FROM providerdata pd
		JOIN kategorijasa kj ON kj.id = pd.jasa_id
		WHERE pd.id=$1`, providerId)

	if errBasicInfo != nil {
		checkErr(errBasicInfo, "Select basic info failed")
	}

	// Get profile pict
	var profileProvider ProviderProfileImage
	errProfilePict := dbmap.SelectOne(&profileProvider,
		`SELECT * FROM providerprofileimage WHERE provider_id=$1`,
		providerId)

	var profilePictUrl string
	var profileBgUrl string

	if errProfilePict != nil {
		profilePictUrl = ""
		profileBgUrl = ""
	} else {
		profilePictUrl = profileProvider.ProfilePict
		profileBgUrl = profileProvider.ProfileBg
	}

	// get images gallery
	var providerGallery []ProviderGallery
	_, errGallery := dbmap.Select(&providerGallery,
		`SELECT * FROM providergallery WHERE provider_id=$1`,
		providerId)

	if errGallery != nil {
	}

	// get price list
	var providerPriceList []ProviderPriceList
	_, errPriceList := dbmap.Select(&providerPriceList,
		`SELECT * FROM providerpricelist WHERE provider_id=$1`,
		providerId)

	if errPriceList != nil {
	}

	// get provider location
	var providerLocation ProviderLatLng
	errLocation := dbmap.SelectOne(&providerLocation,
		`SELECT latitude, longitude FROM providerlocation pl
		WHERE pl.provider_id=$1`, providerId)

	if errLocation != nil {
	}

	c.JSON(200, gin.H{
		"id":           providerBasicInfo.Id,
		"nama":         providerBasicInfo.Nama,
		"alamat":       providerBasicInfo.Alamat,
		"jasa_id":      providerBasicInfo.JasaId,
		"jenis_jasa":   providerBasicInfo.JenisJasa,
		"location":     providerLocation,
		"profile_pict": profilePictUrl,
		"profile_bg":   profileBgUrl,
		"gallery":      providerGallery,
		"price":        providerPriceList,
	})

}

func GetNearProviderForMap(c *gin.Context) {
	// Get all provider that near 2KM from user
	var lat float64
	var long float64
	lat, _ = strconv.ParseFloat(c.Query("lat"), 64)
	long, _ = strconv.ParseFloat(c.Query("long"), 64)

	var searchDistance int64

	if c.Query("distance") == "" {
		searchDistance = 2000
	} else {
		searchDistance, _ = strconv.ParseInt(c.Query("distance"), 0, 64)
	}

	var nearProviderForMap []NearProviderForMap
	_, errNPM := dbmap.Select(&nearProviderForMap,
		`SELECT pd.id as id, pd.nama as nama, kj.id as jasa_id,
		kj.jenis as jenis_jasa, pl.latitude as latitude, pl.longitude as longitude,
		earth_distance(ll_to_earth($1, $2), ll_to_earth(pl.latitude, pl.longitude))
		AS distance
		FROM providerlocation pl
			JOIN providerdata pd on pd.id = pl.provider_id
			JOIN kategorijasa kj on kj.id = pd.jasa_id
		WHERE earth_distance(ll_to_earth($1, $2),
		ll_to_earth(pl.latitude, pl.longitude)) <= $3
		ORDER BY distance ASC`, lat, long, searchDistance)

	var nearProviderByType []NearProviderByType
	_, errNPT := dbmap.Select(&nearProviderByType,
		`SELECT jasa_id, jenis_jasa, COUNT(jasa_id) as count_jasa_provider,
		MIN(distance) as min_distance
		FROM (SELECT kj.id as jasa_id, kj.jenis as jenis_jasa,
			earth_distance(ll_to_earth($1, $2),
			ll_to_earth(pl.latitude, pl.longitude)) as distance
			FROM providerlocation pl
				JOIN providerdata pd on pd.id = pl.provider_id
				JOIN kategorijasa kj on kj.id = pd.jasa_id
			WHERE earth_distance(ll_to_earth($1, $2),
			ll_to_earth(pl.latitude, pl.longitude)) <= $3
			ORDER BY distance ASC) as provider_by_location
		GROUP BY jasa_id, jenis_jasa
		ORDER BY jasa_id ASC`, lat, long, searchDistance)

	if errNPM == nil && errNPT == nil {
		c.JSON(200, gin.H{
			"map":  nearProviderForMap,
			"type": nearProviderByType})
	} else {
		checkErr(errNPM, "Select failed NPM")
		checkErr(errNPT, "Select failed NPT")
	}
}

func GetProvidersByCategory(c *gin.Context) {
	// Get all provider by category
	jasaId := c.Params.ByName("jasa_id")

	var lat float64
	var long float64
	lat, _ = strconv.ParseFloat(c.Query("lat"), 64)
	long, _ = strconv.ParseFloat(c.Query("long"), 64)

	var searchDistance int64

	if c.Query("distance") == "" {
		searchDistance = 2000
	} else {
		searchDistance, _ = strconv.ParseInt(c.Query("distance"), 0, 64)
	}

	var providerByCat []ProviderByCat
	_, err := dbmap.Select(&providerByCat, `
	SELECT pd.id, pd.nama, pl.latitude, pl.longitude, min_price, max_price, rating,
earth_distance(ll_to_earth($1, $2), ll_to_earth(pl.latitude, pl.longitude))
AS distance
FROM providerdata pd join providerlocation pl on pl.provider_id = pd.id
LEFT JOIN (
	SELECT provider_id, MIN(service_price) as min_price, MAX(service_price)
	as max_price
	FROM providerpricelist
	GROUP BY provider_id) pp
ON pp.provider_id = pd.id
LEFT JOIN (
	SELECT provider_id, ((sum_rating + 0.0)/count)::float as rating
	FROM (
		SELECT provider_id, count(*) as count, sum(user_rating) sum_rating
		FROM providerrating group by provider_id) rating_counter) pr
ON pr.provider_id = pd.id
WHERE pd.jasa_id=$3
	AND earth_distance(ll_to_earth($1, $2),
	ll_to_earth(pl.latitude, pl.longitude)) <= $4
ORDER BY distance ASC;
	`, lat, long, jasaId, searchDistance)

	if err == nil {
		/*listProviders := []ListProviderByCat{}
		for _, row := range providerByCat {
			listProviderItem := ListProviderByCat{
				Id:        row.Id,
				Nama:      row.Nama,
				Latitude:  row.Latitude,
				Longitude: row.Longitude,
				MinPrice:  row.MinPrice.Int64,
				MaxPrice:  row.MaxPrice.Int64,
				Rating:    row.Rating.Float64,
				Distance:  row.Distance,
			}
			listProviders = append(listProviders, listProviderItem)
		}*/

		c.JSON(200, gin.H{"data": providerByCat})
	} else {
		checkErr(err, "Select failed")
	}
}

func GetProvidersByKeyword(c *gin.Context) {
	// Get all provider by keyword
}

func getTokenLoginProvider(providerId int64) {

}

func PostSignInProvider(c *gin.Context) {
	var providerAccount ProviderAccount
	c.Bind(&providerAccount)

	var recProviderAccount ProviderAccount
	err := dbmap.SelectOne(&recProviderAccount, `SELECT provider_id FROM provideraccount
		WHERE email=$1 AND password=$2`, providerAccount.Email, providerAccount.Password)

	if err == nil {

		var authTokenProvider AuthTokenProvider

		errAuthToken := dbmap.SelectOne(&authTokenProvider,
			`SELECT id, provider_id, auth_token, expired_date
			FROM authtokenprovider
			WHERE provider_id=$1`, recProviderAccount.ProviderId)

		if errAuthToken != nil {
			authTokenProvider = createAuthTokenProvider(recProviderAccount)
		}

		providerData := getProviderData(recProviderAccount.ProviderId)

		loginAccount := ProviderLoginAccount{
			ProviderId: recProviderAccount.ProviderId,
			FullName: providerData.Nama,
			PhoneNumber: providerData.PhoneNumber,
			Email: recProviderAccount.Email,
			AuthToken: AuthTokenRes{
				Token: authTokenProvider.AuthToken,
				ExpiredDate: authTokenProvider.ExpireDate,
			},
		}

		c.JSON(200, loginAccount)

	} else {
		c.JSON(400, gin.H{"error" : "Account not exists"})
	}
}

func PostCreateProvider(c *gin.Context) {
	// Create new provider
	var providerData ProviderData
	c.Bind(&providerData)

	if insert := db.QueryRow(`INSERT INTO providerdata(nama, email,
		phone_number, jasa_id, alamat, provinsi,
		kabupaten, kelurahan, kode_pos, dokumen, join_date, modified_date)
		VALUES($1, $2, $3, $4, $5, $6, $7,
		$8, $9, $10, $11, $12) RETURNING id`,
		providerData.Nama, providerData.Email, providerData.PhoneNumber,
		providerData.JasaId, providerData.Alamat, providerData.Provinsi,
		providerData.Kabupaten, providerData.Kelurahan, providerData.KodePos,
		providerData.Dokumen, providerData.JoinDate, providerData.ModifiedDate); insert != nil {

		var id int64
		err := insert.Scan(&id)

		insertAccount := db.QueryRow(`INSERT INTO provideraccount(provider_id,
			email, status)
			VALUES($1, $2, $3)`, id, providerData.Email, 0)

		if err == nil && insertAccount != nil {
			content := &ProviderData{
				Id:           id,
				Nama:         providerData.Nama,
				Email:        providerData.Email,
				PhoneNumber:  providerData.PhoneNumber,
				JasaId:       providerData.JasaId,
				Alamat:       providerData.Alamat,
				Provinsi:     providerData.Provinsi,
				Kabupaten:    providerData.Kabupaten,
				Kelurahan:    providerData.Kelurahan,
				KodePos:      providerData.KodePos,
				Dokumen:      providerData.Dokumen,
				JoinDate:     providerData.JoinDate,
				ModifiedDate: providerData.ModifiedDate,
			}
			c.JSON(200, content)
		} else {
			checkErr(err, "Insert failed")
		}
	}
}

func UpdateProviderData(c *gin.Context) {
	// Update provider data
}

func InActiveProvider(c *gin.Context) {
	// Inactive provider
	var providerAccount ProviderAccount
	c.Bind(&providerAccount)

	err := dbmap.SelectOne(&providerAccount,
		`SELECT provider_id FROM provideraccount WHERE provider_id=$1`,
		providerAccount.ProviderId)

	if err == nil {
		if update := db.QueryRow(`UPDATE provideraccount SET status=$1
			WHERE provider_id=$2`, 0,
			providerAccount.ProviderId); update != nil {
			c.JSON(200, gin.H{"status": "update success"})
		} else {
			c.JSON(400, gin.H{"error": "update failed"})
		}

	} else {
		checkErr(err, "Select failed")
	}
}

func ActiveProvider(c *gin.Context) {
	// Active provider
	var providerAccount ProviderAccount
	c.Bind(&providerAccount)

	err := dbmap.SelectOne(&providerAccount, `SELECT provider_id
		FROM provideraccount WHERE provider_id=$1`,
		providerAccount.ProviderId)

	if err == nil {
		if update := db.QueryRow(`UPDATE provideraccount SET status=$1
			WHERE provider_id=$2`, 1,
			providerAccount.ProviderId); update != nil {
			c.JSON(200, gin.H{"status": "update success"})
		} else {
			c.JSON(400, gin.H{"error": "update failed"})
		}

	} else {
		checkErr(err, "Select failed")
	}
}

func PostMyLocationProvider(c *gin.Context) {
	// Post my location for provider
	var providerLocation ProviderLocation
	c.Bind(&providerLocation)

	var recProviderLocation ProviderLocation
	err := dbmap.SelectOne(&recProviderLocation, `SELECT provider_id
		FROM providerlocation WHERE provider_id=$1`,
		providerLocation.ProviderId)

	if err == nil {
		// Already exists
		if update := db.QueryRow(`UPDATE providerlocation SET latitude=$1,
			longitude=$2 WHERE provider_id=$3`,
			providerLocation.Latitude, providerLocation.Longitude,
			providerLocation.ProviderId); update != nil {
			c.JSON(200, gin.H{"status": "success updated my location"})
		}
	} else {
		// Not exists
		if insert := db.QueryRow(`INSERT INTO
			providerlocation(provider_id, latitude, longitude)
		VALUES($1, $2, $3)`,
			providerLocation.ProviderId, providerLocation.Latitude,
			providerLocation.Longitude); insert != nil {
			c.JSON(200, gin.H{"status": "success saved my location"})
		}
	}
}

func PostCreateNewJasa(c *gin.Context) {
	var kategoriJasa KategoriJasa
	c.Bind(&kategoriJasa)

	if insert := db.QueryRow(`INSERT INTO kategorijasa(jenis) VALUES($1)`,
		kategoriJasa.Jenis); insert != nil {
		c.JSON(200, gin.H{"status": "Success create new jenis jasa"})
	}
}

func PostAddProviderPriceList(c *gin.Context) {
	var providerPriceItem ProviderPriceList
	c.Bind(&providerPriceItem)

	var recProvider ProviderData
	err := dbmap.SelectOne(&recProvider, `SELECT * FROM providerdata WHERE id=$1`,
		providerPriceItem.ProviderId)

	if err == nil {
		if insert := db.QueryRow(`INSERT INTO providerpricelist(provider_id,
			service_name, service_price, negotiable)
		VALUES($1, $2, $3, $4)`,
			providerPriceItem.ProviderId,
			providerPriceItem.ServiceName,
			providerPriceItem.ServicePrice,
			providerPriceItem.Negotiable); insert != nil {
			c.JSON(200, gin.H{"status": "Success add new price"})
		}
	} else {
		checkErr(err, "Select failed")
	}

}

func GetProviderPriceList(c *gin.Context) {
	id := c.Params.ByName("provider_id")

	var providerPriceList []ProviderPriceList
	_, err := dbmap.Select(&providerPriceList, `SELECT *
		FROM providerpricelist WHERE provider_id=$1`, id)

	if err == nil {
		c.JSON(200, gin.H{"data": providerPriceList})
	} else {
		checkErr(err, "Select failed")
	}
}

func GetProviderPrice(c *gin.Context) {
	providerId := c.Params.ByName("provider_id")
	id := c.Params.ByName("id")

	var providerPrice ProviderPriceList

	err := dbmap.SelectOne(&providerPrice, `SELECT *
		FROM providerpricelist WHERE id=$1 AND provider_id=$2`, id,
		providerId)

	if err == nil {
		c.JSON(200, providerPrice)
	} else {
		checkErr(err, "Select failed")
	}
}

func UpdateProviderPrice(c *gin.Context) {
	var providerPrice ProviderPriceList
	c.Bind(&providerPrice)

	var recProviderPrice ProviderPriceList
	err := dbmap.SelectOne(&recProviderPrice, `SELECT * FROM providerpricelist
		WHERE provider_id=$1`,
		providerPrice.ProviderId)

	if err == nil {
		if update := db.QueryRow(`UPDATE providerpricelist
			SET service_name=$1, service_price=$2
			WHERE provider_id=$3`, providerPrice.ServiceName, providerPrice.ServicePrice,
			providerPrice.ProviderId); update != nil {
			c.JSON(200, gin.H{"status": "Update success"})
		}
	} else {
		checkErr(err, "Select failed")
	}
}

func PostAddedRating(c *gin.Context) {

	userId := getUserIdFromToken(c)

	var providerRating ProviderRating
	c.Bind(&providerRating)

	var recProvider ProviderData
	errProvider := dbmap.SelectOne(&recProvider, `SELECT * FROM providerdata
		WHERE id=$1`,
		providerRating.ProviderId)

	if errProvider == nil {

		var recProviderRating ProviderRating
		errRating := dbmap.SelectOne(&recProviderRating, `SELECT *
			FROM providerrating
			WHERE user_id=$1 AND provider_id=$2`, userId, providerRating.ProviderId)

		if errRating != nil {
			if insert := db.QueryRow(`INSERT
				INTO providerrating(provider_id, user_id, user_rating)
			VALUES($1, $2, $3)`,
				providerRating.ProviderId,
				userId,
				providerRating.UserRating); insert != nil {
				c.JSON(200, gin.H{"status": "Success give rating"})
			}
		} else {
			c.JSON(400, gin.H{"error": "Only can give rating once"})
		}

	} else {
		checkErr(errProvider, "Select failed")
	}
}

func GetProviderRating(c *gin.Context) {
	providerId := c.Params.ByName("provider_id")

	var providerRating []ProviderRating
	_, err := dbmap.Select(&providerRating, `SELECT * FROM providerrating
		WHERE provider_id=$1`, providerId)

	if err == nil {
		c.JSON(200, gin.H{"data": providerRating})
	} else {
		checkErr(err, "Select failed")
	}

}

func UpdateProviderRating(c *gin.Context) {

	userId := getUserIdFromToken(c)

	var providerRating ProviderRating
	c.Bind(&providerRating)

	var recProviderRating ProviderRating
	err := dbmap.SelectOne(&recProviderRating, `SELECT * FROM providerrating
		WHERE provider_id=$1 AND user_id=$2`,
		providerRating.ProviderId, userId)

	if err == nil {
		if update := db.QueryRow(`UPDATE providerrating SET user_rating=$1
			WHERE provider_id=$2 AND user_id=$3`,
			providerRating.UserRating, providerRating.ProviderId,
			userId); update != nil {
			c.JSON(200, gin.H{"status": "update success"})
		}
	} else {
		checkErr(err, "Select failed")
	}
}

func PostProviderImageGallery(c *gin.Context) {
	var providerGallery ProviderGallery
	c.Bind(&providerGallery)

	var recProvider ProviderData
	err := dbmap.SelectOne(&recProvider, `SELECT * FROM providerdata
		WHERE id=$1`, providerGallery.ProviderId)

	if err == nil {
		if insert := db.QueryRow(`INSERT INTO providergallery(provider_id, image)
			VALUES($1, $2)`, providerGallery.ProviderId, providerGallery.Image); insert != nil {
			c.JSON(200, gin.H{"status": "Success insert new image to gallery"})
		}
	} else {
		checkErr(err, "Select failed")
	}
}

func DeleteImageGallery(c *gin.Context) {
	var providerGallery ProviderGallery
	c.Bind(&providerGallery)

	var recProviderGallery ProviderGallery
	err := dbmap.SelectOne(&recProviderGallery, `SELECT * FROM providergallery
		WHERE id=$1 AND
		provider_id=$2`, providerGallery.Id, providerGallery.ProviderId)

	if err == nil {
		if delete := db.QueryRow(`DELETE FROM providergallery
			WHERE id=$1 AND provider_id=$2`,
			providerGallery.Id, providerGallery.ProviderId); delete != nil {
			c.JSON(200, gin.H{"status": "Delete success"})
		}
	} else {
		checkErr(err, "Select failed")
	}
}

func GetListImageGallery(c *gin.Context) {
	providerId := c.Params.ByName("provider_id")

	var providerGallery []ProviderGallery
	_, err := dbmap.Select(&providerGallery, `SELECT * FROM providergallery
		WHERE provider_id=$1`, providerId)

	if err == nil {
		c.JSON(200, gin.H{"data": providerGallery})
	} else {
		checkErr(err, "Select failed")
	}
}

func GetProfileProvider(c *gin.Context) {
	providerId := c.Params.ByName("provider_id")

	var profileProvider ProviderProfileImage
	err := dbmap.SelectOne(&profileProvider, `SELECT * FROM providerprofileimage
		WHERE provider_id=$1`,
		providerId)

	if err == nil {
		c.JSON(200, profileProvider)
	} else {
		checkErr(err, "Select failed")
	}
}

func PostProfileProvider(c *gin.Context) {
	var profileProvider ProviderProfileImage
	c.Bind(&profileProvider)

	var recProvider ProviderData
	err := dbmap.SelectOne(&recProvider, `SELECT * FROM providerdata WHERE id=$1`,
		profileProvider.ProviderId)

	if err == nil {

		var recProfile ProviderProfileImage
		err := dbmap.SelectOne(&recProfile, `SELECT * FROM providerprofileimage
				WHERE provider_id=$1`, profileProvider.ProviderId)

		if err == nil {
			if update := db.QueryRow(`UPDATE providerprofileimage
					SET profile_pict=$1, profile_bg=$2 WHERE provider_id=$3`,
				profileProvider.ProfilePict,
				profileProvider.ProfileBg,
				profileProvider.ProviderId); update != nil {
				c.JSON(200, gin.H{"status": "update success"})
			}
		} else {
			if insert := db.QueryRow(`INSERT INTO
				providerprofileimage(provider_id, profile_pict, profile_bg)
				VALUES($1, $2, $3)`,
				profileProvider.ProviderId,
				profileProvider.ProfilePict,
				profileProvider.ProfileBg); insert != nil {
				c.JSON(200, gin.H{"status": "Insert new profile pict and bg"})
			}
		}

	} else {
		checkErr(err, "Select failed")
	}
}

func PostNewOrder(c *gin.Context) {

	userId := getUserIdFromToken(c)

	var postTransaction PostTransaction
	c.Bind(&postTransaction)

	fmt.Printf("ID : %d", postTransaction.ProviderId)

	var providerAccount ProviderAccount
	errProvider := dbmap.SelectOne(&providerAccount,
		`SELECT provider_id FROM provideraccount WHERE provider_id=$1`,
		postTransaction.ProviderId)

	var user UserAccount
	errUser := dbmap.SelectOne(&user, `SELECT id FROM useraccount WHERE id=$1`, userId)

	if errProvider != nil {
		c.JSON(400, gin.H{"error": "Penyedia Jasa tidak terdaftar atau tidak aktif"})
	} else if errUser != nil {
		c.JSON(400, gin.H{"error": "User tidak terdaftar"})
	} else {
		if insert := db.QueryRow(`INSERT INTO ordervendor(provider_id,
		user_id,
		destination,
		destination_lat,
		destination_long,
		destination_desc,
		notes,
		payment_method,
		order_date)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
			postTransaction.ProviderId,
			userId,
			postTransaction.Destination,
			postTransaction.DestinationLat,
			postTransaction.DestinationLong,
			postTransaction.DestinationDesc,
			postTransaction.Notes,
			postTransaction.PaymentMethod,
			postTransaction.OrderDate); insert != nil {

			var orderId int64
			err := insert.Scan(&orderId)

			if err == nil {

				// insert order journey
				/* status
				0 = Waiting confirmation
				1 = Confirmed
				2 = On the way
				3 = Reach destination
				4 = On working
				5 = Complete
				6 = Cancel
				*/
				orderVendorJourney := OrderVendorJourney{
					OrderId: orderId,
					Status:  0,
				}

				db.QueryRow(`INSERT INTO ordervendorjourney(order_id, status)
			VALUES($1, $2)`, orderVendorJourney.OrderId, orderVendorJourney.Status)

				// insert order tracking
				orderTracking := OrderVendorTracking{
					OrderId:          orderId,
					CurrentLatitude:  0,
					CurrentLongitude: 0,
				}

				db.QueryRow("INSERT INTO ordervendortracking(order_id, latitude, longitude) VALUES($1, $2, $3)",
					orderTracking.OrderId, orderTracking.CurrentLatitude, orderTracking.CurrentLongitude)

				for i := 0; i < len(postTransaction.Data); i++ {
					orderVendorDetail := &OrderVendorDetail{
						OrderId:      orderId,
						JasaId:       postTransaction.Data[i].JasaId,
						ServiceName:  postTransaction.Data[i].ServiceName,
						ServicePrice: postTransaction.Data[i].ServicePrice,
						Qty:          postTransaction.Data[i].Qty,
						ModifiedDate: postTransaction.Data[i].ModifiedDate,
					}

					db.QueryRow(`INSERT INTO ordervendordetail(order_id,
					jasa_id,
					service_name,
					service_price,
					qty,
					modified_date)
					VALUES($1, $2, $3, $4, $5, $6)`,
						orderVendorDetail.OrderId,
						orderVendorDetail.JasaId,
						orderVendorDetail.ServiceName,
						orderVendorDetail.ServicePrice,
						orderVendorDetail.Qty,
						orderVendorDetail.ModifiedDate)
				}

				// send notification to vendor

				c.JSON(200, gin.H{"status": "Success order"})
			} else {
				checkErr(err, "Insert transaction failed")
			}
		}
	}
}

func PostNewOrderJourney(c *gin.Context) {
	var orderVendorJourney OrderVendorJourney
	c.Bind(&orderVendorJourney)

	if insert := db.QueryRow(`INSERT INTO ordervendorjourney(order_id, status)
	VALUES($1, $2)`,
		orderVendorJourney.OrderId, orderVendorJourney.Status); insert != nil {

		c.JSON(200, gin.H{"status": "success"})
	} else {
		c.JSON(400, gin.H{"error": "Failed update order status"})
	}

}

func UpdateOrderTracking(c *gin.Context) {
	var orderVendorTracking OrderVendorTracking
	c.Bind(&orderVendorTracking)

	var recOrderVendorTracking OrderVendorTracking
	err := dbmap.SelectOne(&recOrderVendorTracking, `SELECT id,
		order_id FROM ordervendortracking
		WHERE id=$1 AND order_id=$2`, orderVendorTracking.Id,
		orderVendorTracking.OrderId)

	if err == nil {
		if update := db.QueryRow(`UPDATE ordervendortracking
			SET latitude=$1, longitude=$2 WHERE
			id=$3 AND order_id=$4`, orderVendorTracking.CurrentLatitude,
			orderVendorTracking.CurrentLongitude,
			recOrderVendorTracking.Id, orderVendorTracking.OrderId); update != nil {
			c.JSON(200, gin.H{"status": "Success update current vendor location"})
		} else {
			c.JSON(400, gin.H{"error": "Failed update tracking record"})
		}
	} else {
		c.JSON(400, gin.H{"error": "Record not found"})
	}
}

func authenticateEmailAccount(userAccount UserAccount) UserAccount {
	var recAuthAccount UserAccount
	errAuthAccount := dbmap.SelectOne(&recAuthAccount,
		`SELECT id, email, auth_mode FROM useraccount
		WHERE email=$1 AND password=$2`, userAccount.Email,
		userAccount.Password)

	if errAuthAccount == nil {
		return recAuthAccount;
	} else {
		return UserAccount{}
	}
}

func authenticateSocialAccount(userAccount UserAccount) UserAccount {
	var recAuthAccount UserAccount
	errAuthAccount := dbmap.SelectOne(&recAuthAccount,
		`SELECT id, email, auth_mode FROM useraccount
		WHERE email=$1`, userAccount.Email)

	if errAuthAccount == nil {
		return recAuthAccount;
	} else {
		return UserAccount{}
	}
}

func isAccountExists(userAccount UserAccount) bool {
	var recAuthAccount UserAccount
	errAuthAccount := dbmap.SelectOne(&recAuthAccount,
		`SELECT id FROM useraccount WHERE email=$1`, userAccount.Email)

	if errAuthAccount == nil {
		return true;
	}

	return false;
}

func PostSignInEmail(c *gin.Context) {
	var userAccount UserAccount
	c.Bind(&userAccount)

	loginWithRegisteredAccount(userAccount, c)
}

func loginWithRegisteredAccount(userAccount UserAccount, c *gin.Context) {
	var recAuthAccount UserAccount

	if userAccount.AuthMode == "email" {
		recAuthAccount = authenticateEmailAccount(userAccount)
	} else {
		recAuthAccount = authenticateSocialAccount(userAccount)
	}

	if recAuthAccount.Email != "" {

		var authToken AuthToken

		errAuthToken := dbmap.SelectOne(&authToken,
			`SELECT id, user_id, auth_token, expired_date
			FROM authtoken
			WHERE user_id=$1`, recAuthAccount.Id)

		if errAuthToken != nil {
			authToken = createAuthToken(recAuthAccount)
		} else {
			if authToken.ExpireDate <= time.Now().Unix() {
				removeExpiredToken(authToken.Id)
				authToken = createAuthToken(recAuthAccount)
			}
		}

		userProfile := getUserProfile(recAuthAccount.Id)

		loginAccount := LoginAccount{
			UserId: recAuthAccount.Id,
			FullName: userProfile.FullName,
			PhoneNumber: userProfile.PhoneNumber,
			Email: recAuthAccount.Email,
			AuthMode: recAuthAccount.AuthMode,
			AuthToken: AuthTokenRes{
				Token: authToken.AuthToken,
				ExpiredDate: authToken.ExpireDate,
			},
		}

		c.JSON(200, loginAccount)

	} else {
		c.JSON(400, gin.H{"error" : "Account not found"})
	}
}

func getUserProfile(userId int64) UserProfile {
	var userProfile UserProfile

	dbmap.SelectOne(&userProfile, `SELECT * FROM userprofile WHERE user_id=$1`, userId)

	return userProfile
}

func getProviderData(providerId int64) ProviderData {
	var providerData ProviderData

	dbmap.SelectOne(&providerData, `SELECT * FROM providerdata WHERE id=$1`, providerId)

	return providerData
}

func createAuthTokenProvider(recProviderAccount ProviderAccount) AuthTokenProvider {
	expiredTime := time.Now().Add(time.Hour * 24).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": recProviderAccount.Id,
		"email": recProviderAccount.Email,
		"exp": expiredTime,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, errCreateToken := token.SignedString(mySigningKey)

	if errCreateToken == nil {
		if insert := db.QueryRow(
			`INSERT INTO authtokenprovider(provider_id, auth_token, expired_date)
			VALUES($1, $2, $3) RETURNING ID`, recProviderAccount.ProviderId, tokenString, expiredTime);
		insert != nil {

			var id int64

			insert.Scan(&id)

			return AuthTokenProvider{
				AuthToken: tokenString,
				ExpireDate: expiredTime,
			}
		} else {
			return AuthTokenProvider{}
		}
	} else {
		return AuthTokenProvider{}
	}
}

func createAuthToken(recAuthAccount UserAccount) AuthToken {
	expiredTime := time.Now().Add(time.Hour * 24).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": recAuthAccount.Id,
		"email": recAuthAccount.Email,
		"exp": expiredTime,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, errCreateToken := token.SignedString(mySigningKey)

	if errCreateToken == nil {
		if insert := db.QueryRow(
			`INSERT INTO authtoken(user_id, auth_token, expired_date)
			VALUES($1, $2, $3) RETURNING ID`, recAuthAccount.Id, tokenString, expiredTime);
		insert != nil {

			var id int64

			insert.Scan(&id)

			return AuthToken{
				AuthToken: tokenString,
				ExpireDate: expiredTime,
			}
		} else {
			return AuthToken{}
		}
	} else {
		return AuthToken{}
	}
}

func PostSignUpEmail(c *gin.Context) {
	var userAccount UserAccount
	c.Bind(&userAccount)

	if !isAccountExists(userAccount) {
		joinDate := time.Now().Add(time.Hour * 24).Unix()

		if insert := db.QueryRow(`INSERT INTO useraccount(email, password, auth_mode,
		device_token, join_date) VALUES($1, $2, $3, $4, $5) RETURNING id`,
			userAccount.Email, userAccount.Password, userAccount.AuthMode,
			userAccount.DeviceToken, joinDate);
		insert != nil {

			loginWithRegisteredAccount(userAccount, c)

		}
	} else {
		c.JSON(400, gin.H{"error" : "Account already exists"})
	}
}

func PostAuthSocial(c *gin.Context) {
	var userAccount UserAccount
	c.Bind(&userAccount)

	if !isAccountExists(userAccount) {
		// sign up
		joinDate := time.Now().Add(time.Hour * 24).Unix()

		if insert := db.QueryRow(`INSERT INTO useraccount(email, password, auth_mode,
		device_token, join_date) VALUES($1, $2, $3, $4, $5) RETURNING id`,
			userAccount.Email, userAccount.Password, userAccount.AuthMode,
			userAccount.DeviceToken, joinDate);
		insert != nil {

			loginWithRegisteredAccount(userAccount, c)

		}
	} else {
		// sign in
		loginWithRegisteredAccount(userAccount, c)
	}
}

func PutProfileUpdate(c *gin.Context) {
	var userProfile UserProfile
	c.Bind(&userProfile)

	userId := getUserIdFromToken(c)

	if userId != -1 {
		var recUserProfile UserProfile
		err := dbmap.SelectOne(&recUserProfile, `SELECT * FROM userprofile WHERE user_id=$1`, userId)

		if err == nil {
			if update := db.QueryRow(`UPDATE userprofile SET full_name=$1, address=$2,
			dob=$3, phone_number=$4 WHERE user_id=$5`,
				userProfile.FullName, userProfile.Address,
				userProfile.DOB, userProfile.PhoneNumber, userId);
			update != nil {

				c.JSON(200, gin.H{"status" : "Success update data",
					"data" : UserProfile{
						UserId: userId,
						FullName: userProfile.FullName,
						Address: userProfile.Address,
						DOB: userProfile.DOB,
						PhoneNumber: userProfile.PhoneNumber,
					},
				})
			}
		} else {
			if insert := db.QueryRow(`INSERT INTO userprofile(user_id, full_name, address,
				dob, phone_number) VALUES($1, $2, $3, $4, $5)`, userId,
				userProfile.FullName, userProfile.Address,
				userProfile.DOB, userProfile.PhoneNumber);
			insert != nil {

				c.JSON(200, gin.H{"status" : "Success update data",
					"data" : UserProfile{
						UserId: userId,
						FullName: userProfile.FullName,
						Address: userProfile.Address,
						DOB: userProfile.DOB,
						PhoneNumber: userProfile.PhoneNumber,
					},
				})
			}
		}
	}
}

func PutDeviceTokenUpdate(c *gin.Context) {
	var userAccount UserAccount
	c.Bind(&userAccount)

	userId := getUserIdFromToken(c)

	if userId != -1 {
		if update := db.QueryRow(`UPDATE useraccount SET device_token=$1 WHERE id=$2`,
			userAccount.DeviceToken, userId);
		update != nil {
			c.JSON(200, gin.H{"success" : "Device token updated" })
		}
	} else {
		c.JSON(400, gin.H{"error" : "Account not found"})
	}
}

func getUserIdFromToken(c *gin.Context) int64 {
	tokenStr := getTokenFromHeader(c)

	var authToken AuthToken
	err := dbmap.SelectOne(&authToken, `SELECT user_id FROM authtoken WHERE auth_token=$1`, tokenStr)

	if err == nil {
		return authToken.UserId
	} else {
		return -1
	}
}

func GetUserProfile(c *gin.Context) {
	userId := getUserIdFromToken(c)

	userProfile := getUserProfile(userId)

	c.JSON(200, userProfile)
}
