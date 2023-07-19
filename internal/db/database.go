package db

import (
	"database/sql"
	"embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	_ "github.com/asg017/sqlite-vss/bindings/go"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sashabaranov/go-openai"

	"github.com/dleviminzi/chatrr/internal/models"
)

//go:embed migrations
var migrationFiles embed.FS

type DatabaseConnection struct {
	DB *sql.DB
}

func NewDatabaseConnection() *DatabaseConnection {
	dbPath, err := getDbPath()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	dbConn := &DatabaseConnection{DB: db}

	err = dbConn.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	err = dbConn.ApplyMigrations()
	if err != nil {
		log.Fatal(err)
	}

	return dbConn
}

// GetCurrentVersion retrieves the current schema version from the database.
// It queries the 'schema_migrations' table for the maximum version number, which represents
// the latest applied migration. If an error occurs during this process, it will be returned
// along with a zero value for the version.
func (d *DatabaseConnection) GetCurrentVersion() (int, error) {
	var version int
	row := d.DB.QueryRow("SELECT MAX(version) FROM schema_migrations")
	err := row.Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("could not get current schema version: %w", err)
	}
	return version, nil
}

// ApplyMigrations checks the current database schema version and applies all migrations
// that have a higher version number. The migrations are stored as .sql files in an
// embedded 'migrations' directory. The files should be named as an increasing version number
// followed by '.sql', like '1.sql', '2.sql', etc.
func (d *DatabaseConnection) ApplyMigrations() error {
	currentVersion, err := d.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("could not apply migrations: %w", err)
	}

	files, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("could not read migration directory: %w", err)
	}

	for _, file := range files {
		version, err := strconv.Atoi(strings.TrimSuffix(file.Name(), ".sql"))
		if err != nil || version <= currentVersion { // skip if invalid or unnecessary
			continue
		}

		migration, err := migrationFiles.ReadFile("migrations/" + file.Name())
		if err != nil {
			return fmt.Errorf("could not read file %s: %w", file.Name(), err)
		}

		_, err = d.DB.Exec(string(migration))
		if err != nil {
			return fmt.Errorf("could not execute migration %s: %w", file.Name(), err)
		}

		_, err = d.DB.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version)
		if err != nil {
			return fmt.Errorf("could not update schema version to %d: %w", version, err)
		}
	}

	return nil
}

// Initialize sets up the schema_migrations table for version tracking.
func (d *DatabaseConnection) Initialize() error {
	_, err := d.DB.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)")
	if err != nil {
		return fmt.Errorf("could not create schema_migrations table: %w", err)
	}

	// If there's no entry in the schema_migrations table, it means it's the first time the app runs.
	// So, we insert a row with version 0 into the table.
	row := d.DB.QueryRow("SELECT COUNT(version) FROM schema_migrations")
	var count int
	err = row.Scan(&count)
	if err != nil {
		return fmt.Errorf("could not check the number of schema versions: %w", err)
	}

	if count == 0 {
		_, err = d.DB.Exec("INSERT INTO schema_migrations (version) VALUES (0)")
		if err != nil {
			return fmt.Errorf("could not initialize schema version to 0: %w", err)
		}
	}

	return nil
}

func (d DatabaseConnection) CreateConversationMemories(memoryEmbeddings [][]float32, conversationId int, conversationFragment []openai.ChatCompletionMessage) error {
	for _, embedding := range memoryEmbeddings {
		if err := d.CreateConversationMemory(embedding, conversationId, conversationFragment); err != nil {
			return err
		}
	}
	return nil
}

func (d DatabaseConnection) CreateConversationMemory(memoryEmbedding []float32, conversationId int, conversationFragment []openai.ChatCompletionMessage) error {
	bytePromptEmbedding := byteEmbedding(memoryEmbedding)

	strConversationFragment, err := stringifyConversationFragment(conversationFragment)
	if err != nil {
		return err
	}

	tx, err := d.DB.Begin()
	if err != nil {
		log.Fatal(err)
	}

	f, err := tx.Exec("INSERT INTO conversation_fragments (conversation_id, conversation_fragment) VALUES (?, ?)", conversationId, strConversationFragment)
	if err != nil {
		log.Fatal(err)
	}
	id, err := f.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec("INSERT INTO conversation_fragment_embeddings(rowid, embedding) values (?, ?)", id, bytePromptEmbedding)
	if err != nil {
		panic(err)
	}

	return tx.Commit()
}

func (d DatabaseConnection) GetConversationMemories(promptEmbedding []float32) ([]models.RecalledMemory, error) {
	// TODO: add cosine similarity calculation to vss extension and return that as well for decision making
	query := `
		with matches as (
			select rowid, embedding, vss_cosine_similarity(embedding, ?1) similarity
			from conversation_fragment_embeddings 
			where vss_search(embedding, ?1)
			limit 20
		), final as (
			select
			cf.rowid,
			cf.conversation_id,
			cf.conversation_fragment,
			m.similarity
			from matches m
			left join conversation_fragments cf on cf.id = m.rowid
		)
		select conversation_fragment, max(similarity) 
		from final 
		group by conversation_fragment
		order by similarity desc
		limit 5
	`

	bytePromptEmbedding := byteEmbedding(promptEmbedding)

	rows, err := d.DB.Query(query, bytePromptEmbedding)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []models.RecalledMemory
	// FIXME: bug in vss sqlite extension that makes this explode when the result contains no rows
	for rows.Next() {
		var cm string
		var sim float32

		if err := rows.Scan(&cm, &sim); err != nil {
			return nil, err
		}

		memories = append(memories, models.RecalledMemory{
			ConversationFragment: cm,
			SimilarityScore:      sim,
		})
	}

	return memories, nil
}

func byteEmbedding(embedding []float32) []byte {
	byteEmbedding := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(byteEmbedding[i*4:], math.Float32bits(v))
	}
	return byteEmbedding
}

func stringifyConversationFragment(conversationFragment []openai.ChatCompletionMessage) (string, error) {
	b, err := json.Marshal(conversationFragment)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func getDbPath() (string, error) {
	if os.Getenv("ENV") == "dev" {
		return "internal/db/chatrr.db", nil
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	var dbDir string
	switch runtime.GOOS {
	case "windows":
		dbDir = filepath.Join(usr.HomeDir, "AppData", "Local", "Chatrr")
	case "darwin":
		dbDir = filepath.Join(usr.HomeDir, "Library", "Application Support", "Chatrr")
	default: // Assume a Unix-like system
		dbDir = filepath.Join(usr.HomeDir, ".chatrr")
	}

	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return "", err
		}
	}

	return filepath.Join(dbDir, "chatrr.db"), nil
}
