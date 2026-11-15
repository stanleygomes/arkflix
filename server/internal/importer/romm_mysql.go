package importer

import (
	"context"
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

// openMySQL abre o MariaDB do RomM e devolve uma transação READ ONLY: o
// import só faz SELECT, e o banco recusa qualquer escrita nela.
func openMySQL(ctx context.Context, dsn string) (RomMSource, func(), error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, nil, err
	}
	cfg.ParseTime = true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, nil, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	return SQLRomM{Q: tx}, func() { tx.Rollback(); db.Close() }, nil
}
