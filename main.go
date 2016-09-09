package main

import (
	"github.com/gin-gonic/gin"
	"os"
	"database/sql"
	"gopkg.in/gorp.v1"
	"log"

	_ "github.com/lib/pq"
	"strconv"
)

var db = initDb()
var dbmap = initDbmap()

func initDb() *sql.DB {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))

	checkErr(err, "Failed open db")
	return db;
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
		v1.GET("/providers", GetProviders)
		v1.GET("/providers/near", GetNearProviderForMap)
		v1.GET("/providers/cat_near", GetNearProviderByType)
		v1.GET("/providers/search", GetProvidersByKeyword)
		v1.GET("/provider/cat/:cat_id", GetProvidersByCategory)
		v1.GET("/provider/data/:id", GetProvider)
		v1.POST("/provider/create", PostCreateProvider)
		v1.PUT("/provider/edit/:provider_id", UpdateProviderData)
		v1.PUT("/provider/inactive", InActiveProvider)
		v1.PUT("/provider/active", ActiveProvider)
		v1.POST("/provider/mylocation", PostMyLocationProvider)
		v1.POST("/jasa/create", PostCreateNewJasa)
	}
	r.Run(GetPort())

}

func GetPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "4747"
	}
	return ":" + port
}

type ProviderAccount struct {
	Id int64 `db:"id" json:"id"`
	ProviderId int64 `db:"provider_id" json:"provider_id"`
	Email string `db:"email" json:"email"`
	Token string `db:"token" json:"token"`
	DeviceId string `db:"device_id" json:"device_id"`
	Status int8 `db:"status" json:"status"`
}

type ProviderData struct {
	Id           int64 `db:"id" json:"id"`
	Nama         string `db:"nama" json:"nama"`
	Email        string `db:"email" json:"email"`
	PhoneNumber  string `db:"phone_number" json:"phone_number"`
	JasaId       int8 `db:"jasa_id" json:"jasa_id"`
	Alamat       string `db:"alamat" json:"alamat"`
	Provinsi     string `db:"provinsi" json:"provinsi"`
	Kabupaten    string `db:"kabupaten" json:"kabupaten"`
	Kelurahan    string `db:"kelurahan" json:"kelurahan"`
	KodePos      string `db:"kode_pos" json:"kode_pos"`
	Dokumen      string `db:"dokumen" json:"dokumen"`
	JoinDate     int64 `db:"join_date" json:"join_date"`
	ModifiedDate int64 `db:"modified_date" json:"modified_date"`
}

type ProviderLocation struct {
	Id int64 `db:"id" json:"id"`
	ProviderId int64 `db:"provider_id" json:"provider_id"`
	Latitude float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
}

type KategoriJasa struct {
	Id int64 `db:"id" json:"id"`
	Jenis string `db:"jenis" json:"jenis"`
}

type UserLocation struct {
	UserId int64 `db:"user_id" json:"user_id"`
	Latitude float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
}

type NearProviderForMap struct {
	Id int64 `db:"id" json:"id"`
	Nama string `db:"nama" json:"nama"`
	JasaId int64 `db:"jasa_id" json:"jasa_id"`
	JenisJasa string `db:"jenis_jasa" json:"jenis_jasa"`
	Latitude float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
	Distance float64 `db:"distance" json:"distance"`
}

type NearProviderByType struct {
	JasaId int64 `db:"jasa_id" json:"jasa_id"`
	JenisJasa string `db:"jenis_jasa" json:"jenis_jasa"`
	CountJasaProvider int8	`db:"count_jasa_provider" json:"count_jasa_provider"`
}

type User struct {
	Id        int64 `db:"id" json:"id"`
	Firstname string `db:"firstname" json:"firstname"`
	Lastname  string `db:"lastname" json:"lastname"`
}


func GetProviders(c *gin.Context) {
	// Get all list providers
	type Users []User
	var users = Users{
		User{Id: 1, Firstname: "Oliver", Lastname: "Queen"},
		User{Id: 2, Firstname: "Malcom", Lastname: "Merlyn"},
	}
	c.JSON(200, users)
}

func GetProvider(c *gin.Context) {
	// Get provider by id
}

func GetNearProviderForMap(c *gin.Context) {
	// Get all provider that near 2KM from user
	var lat float64
	var long float64
	lat, _ = strconv.ParseFloat(c.Query("lat"), 64)
	long, _ = strconv.ParseFloat(c.Query("long"), 64)

	var nearProviderForMap []NearProviderForMap
	_, errNPM := dbmap.Select(&nearProviderForMap,
		`SELECT pd.id as id, pd.nama as nama, kj.id as jasa_id, kj.jenis as jenis_jasa,
		pl.latitude as latitude, pl.longitude as longitude,
		earth_distance(ll_to_earth($1, $2), ll_to_earth(pl.latitude, pl.longitude)) AS distance
		FROM providerlocation pl
			JOIN providerdata pd on pd.id = pl.provider_id
			JOIN kategorijasa kj on kj.id = pd.jasa_id
		WHERE earth_distance(ll_to_earth($1, $2), ll_to_earth(pl.latitude, pl.longitude)) <= 2000
		ORDER BY distance ASC`, lat, long)

	var nearProviderByType []NearProviderByType
	_, errNPT := dbmap.Select(&nearProviderByType,
		 `SELECT jasa_id, jenis_jasa, count(jasa_id) as count_jasa_provider
		 FROM (SELECT kj.id as jasa_id, kj.jenis as jenis_jasa,
		 	earth_distance(ll_to_earth($1, $2), ll_to_earth(pl.latitude, pl.longitude)) as distance
		 	FROM providerlocation pl
		 		JOIN providerdata pd on pd.id = pl.provider_id
		 		JOIN kategorijasa kj on kj.id = pd.jasa_id
		 	WHERE earth_distance(ll_to_earth($1, $2), ll_to_earth(pl.latitude, pl.longitude)) <= 2000
		 	ORDER BY distance ASC) as provider_by_location
		 GROUP BY jasa_id, jenis_jasa
		 ORDER BY jasa_id ASC`, lat, long)

	if errNPM == nil && errNPT == nil {
		c.JSON(200, gin.H{
			"map" : nearProviderForMap,
			"type" : nearProviderByType})
	} else {
		checkErr(errNPM, "Select failed NPM")
		checkErr(errNPT, "Select failed NPT")
	}
}

func GetNearProviderByType(c *gin.Context) {
	// Get all provider by type?
}

func GetProvidersByCategory(c *gin.Context) {
	// Get all provider by category
}

func GetProvidersByKeyword(c *gin.Context) {
	// Get all provider by keyword
}

func PostCreateProvider(c *gin.Context) {
	// Create new provider
	var providerData ProviderData
	c.Bind(&providerData)

	if insert := db.QueryRow(`INSERT INTO providerdata(nama, email, phone_number, jasa_id, alamat, provinsi,
		kabupaten, kelurahan, kode_pos, dokumen, join_date, modified_date) VALUES($1, $2, $3, $4, $5, $6, $7,
		$8, $9, $10, $11, $12) RETURNING id`,
		providerData.Nama, providerData.Email, providerData.PhoneNumber, providerData.JasaId,
		providerData.Alamat, providerData.Provinsi, providerData.Kabupaten, providerData.Kelurahan,
		providerData.KodePos, providerData.Dokumen, providerData.JoinDate, providerData.ModifiedDate);
	insert != nil {

		var id int64
		err := insert.Scan(&id)

		insertAccount := db.QueryRow(`INSERT INTO provideraccount(provider_id, email, status)
			VALUES($1, $2, $3)`, id, providerData.Email, 0)

		if err == nil && insertAccount != nil {
			content := &ProviderData {
				Id: id,
				Nama: providerData.Nama,
				Email: providerData.Email,
				PhoneNumber: providerData.PhoneNumber,
				JasaId: providerData.JasaId,
				Alamat: providerData.Alamat,
				Provinsi: providerData.Provinsi,
				Kabupaten: providerData.Kabupaten,
				Kelurahan: providerData.Kelurahan,
				KodePos: providerData.KodePos,
				Dokumen: providerData.Dokumen,
				JoinDate: providerData.JoinDate,
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
	c.Bind(&providerAccount);

	err := dbmap.SelectOne(&providerAccount, "SELECT provider_id FROM provideraccount WHERE provider_id=$1",
		providerAccount.ProviderId)

	if err == nil {
		if update := db.QueryRow("UPDATE provideraccount SET status=$1 WHERE provider_id=$2", 0,
			providerAccount.ProviderId);

		update != nil {
			c.JSON(200, gin.H{"status":"update success"})
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
	c.Bind(&providerAccount);

	err := dbmap.SelectOne(&providerAccount, "SELECT provider_id FROM provideraccount WHERE provider_id=$1",
		providerAccount.ProviderId)

	if err == nil {
		if update := db.QueryRow("UPDATE provideraccount SET status=$1 WHERE provider_id=$2", 1,
			providerAccount.ProviderId);
			update != nil {
			c.JSON(200, gin.H{"status":"update success"})
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
	err := dbmap.SelectOne(&recProviderLocation, "SELECT provider_id FROM providerlocation WHERE provider_id=$1",
		providerLocation.ProviderId)

	if err == nil {
		// Already exists
		if update := db.QueryRow("UPDATE providerlocation SET latitude=$1, longitude=$2 WHERE provider_id=$3",
			providerLocation.Latitude, providerLocation.Longitude, providerLocation.ProviderId);
		update != nil {
			c.JSON(200, gin.H{"status":"success updated my location"})
		}
	} else {
		// Not exists
		if insert := db.QueryRow(`INSERT INTO providerlocation(provider_id, latitude, longitude)
		VALUES($1, $2, $3)`,
			providerLocation.ProviderId, providerLocation.Latitude, providerLocation.Longitude);
		insert != nil {
			c.JSON(200, gin.H{"status":"success saved my location"})
		}
	}
}

func PostCreateNewJasa(c *gin.Context) {
	var kategoriJasa KategoriJasa
	c.Bind(&kategoriJasa)

	if insert := db.QueryRow("INSERT INTO kategorijasa(jenis) VALUES($1)", kategoriJasa.Jenis);
		insert != nil {
		c.JSON(200, gin.H{"status":"Success create new jenis jasa"})
	}
}

