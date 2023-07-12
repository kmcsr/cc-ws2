
package main

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

type MySQLAPI struct {
	DB *sql.DB
}

var _ API = (*MySQLAPI)(nil)

const mysqlDeadLockCode = 1213

func NewMySQLAPI(username string, passwd string, address string, database string)(v *MySQLAPI, err error){
	v = &MySQLAPI{}

	if v.DB, err = sql.Open("mysql",
		fmt.Sprintf("%s:%s@%s/%s?parseTime=true", username, passwd, address, database)); err != nil {
		return
	}
	v.DB.SetConnMaxLifetime(time.Minute * 3)
	v.DB.SetMaxOpenConns(128)
	v.DB.SetMaxIdleConns(16)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
	defer cancel()
	if err = v.DB.PingContext(ctx); err != nil {
		return
	}

	if err = v.createAndLogRootTokenIfNotExists(); err != nil {
		return
	}

	return
}

func (v *MySQLAPI)createAndLogRootTokenIfNotExists()(err error){
	const queryCmd = "SELECT 1 FROM tokens" +
		" WHERE (`expiration` IS NULL OR CONVERT_TZ(`expiration`,@@session.time_zone,'+00:00')>=NOW())" +
		" AND `root`=TRUE"
	const insertCmd = "INSERT INTO tokens (`token`, `root`, `expiration`)" +
		" VALUES (?, TRUE, NULL)"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	var ok bool
	if err = v.DB.QueryRowContext(ctx, queryCmd).Scan(&ok); err == nil && ok {
		return
	}

	loger.Warn("Root token is not exists, creating one...")

	token, err := generateToken()
	if err != nil {
		return
	}

	if _, err = v.DB.ExecContext(ctx, insertCmd, token); err != nil {
		return
	}

	loger.Warn("****************************************************************")
	loger.Warnf("new_root_token=%s", cliTokenPrefix + token)
	loger.Warn("****************************************************************")
	return
}

func (v *MySQLAPI)QueryContext(ctx context.Context, cmd string, args ...any)(rows *sql.Rows, err error){
	loger.Debugf("Query sql cmd: %s\n  args: %v", cmd, args)
	for {
		if rows, err = v.DB.QueryContext(ctx, cmd, args...); err != nil {
			if e, ok := err.(*mysql.MySQLError); ok {
				switch e.Number {
				case mysqlDeadLockCode:
					continue
				}
			}
		}
		return
	}
}

func (v *MySQLAPI)NewCliToken(expiration *time.Time)(token string, err error){
	const insertCmd = "INSERT INTO tokens (`token`, `expiration`)" +
		" VALUES (?, ?)"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	if token, err = generateToken(); err != nil {
		return
	}

	tx, err := v.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err = execTx(tx, insertCmd, token, expiration); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}

	token = cliTokenPrefix + token
	return
}

func (v *MySQLAPI)NewDaemonToken(server string, expiration *time.Time)(token string, err error){
	const insertCmd = "INSERT INTO daemon_tokens (`token`, `server`, `expiration`)" +
		" VALUES (?, ?, ?)"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	if token, err = generateToken(); err != nil {
		return
	}

	tx, err := v.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err = execTx(tx, insertCmd, token, server, expiration); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}

	token = daemonTokenPrefix + token
	return
}

func (v *MySQLAPI)RemoveCliToken(token string)(err error){
	const deleteCmd = "DELETE FROM tokens" +
		" WHERE `token`=?"

	var ok bool
	if token, ok = preProcessCliToken(token); !ok {
		return TokenNotExistsErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	tx, err := v.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err = execTx(tx, deleteCmd, token); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)RemoveDaemonToken(token string)(err error){
	const deleteCmd = "DELETE FROM daemon_tokens" +
		" WHERE `token`=?"

	if len(token) != daemonTokenLen || token[:len(daemonTokenPrefix)] != daemonTokenPrefix {
		return TokenNotExistsErr
	}
	token = token[len(daemonTokenPrefix):]

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	tx, err := v.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err = execTx(tx, deleteCmd, token); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)ListTokens()(tokens []Token, err error){
	const queryCmd = "SELECT `token`,`root`," +
		"CONVERT_TZ(`expiration`,@@session.time_zone,'+00:00')" +
		" FROM tokens"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	var rows *sql.Rows
	if rows, err = v.QueryContext(ctx, queryCmd); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var (
			tk Token
			expiration sql.NullTime
		)
		if err = rows.Scan(&tk.Token, &tk.Root, &expiration); err != nil {
			return
		}
		tk.Token = cliTokenPrefix + tk.Token
		if expiration.Valid {
			tk.Expiration = &expiration.Time
		}
		tokens = append(tokens, tk)
	}
	if err = rows.Err(); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)ListDaemonTokens()(tokens []DaemonToken, err error){
	const queryCmd = "SELECT `token`,`server`," +
		"CONVERT_TZ(`expiration`,@@session.time_zone,'+00:00')" +
		" FROM daemon_tokens"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	var rows *sql.Rows
	if rows, err = v.QueryContext(ctx, queryCmd); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var (
			tk DaemonToken
			expiration sql.NullTime
		)
		if err = rows.Scan(&tk.Token, &tk.Server, &expiration); err != nil {
			return
		}
		tk.Token = daemonTokenPrefix + tk.Token
		if expiration.Valid {
			tk.Expiration = &expiration.Time
		}
		tokens = append(tokens, tk)
	}
	if err = rows.Err(); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)AuthCli(token string)(ok bool){
	const queryCmd = "SELECT 1 FROM tokens" +
		" WHERE (`expiration` IS NULL OR CONVERT_TZ(`expiration`,@@session.time_zone,'+00:00')>=NOW())" +
		" AND `token`=?"

	if token, ok = preProcessCliToken(token); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	if err := v.DB.QueryRowContext(ctx, queryCmd, token).Scan(&ok); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)AuthDaemon(token string, server string)(ok bool){
	const queryCmd = "SELECT 1 FROM daemon_tokens" +
		" WHERE (`expiration` IS NULL OR CONVERT_TZ(`expiration`,@@session.time_zone,'+00:00')>=NOW())" +
		" AND `token`=? AND `server`=?"

	if len(token) != daemonTokenLen || token[:len(daemonTokenPrefix)] != daemonTokenPrefix {
		return false
	}
	token = token[len(daemonTokenPrefix):]

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	if err := v.DB.QueryRowContext(ctx, queryCmd, token, server).Scan(&ok); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)CheckRootToken(rtToken string)(ok bool){
	const queryCmd = "SELECT `root` FROM tokens" +
		" WHERE (`expiration` IS NULL OR CONVERT_TZ(`expiration`,@@session.time_zone,'+00:00')>=NOW())" +
		" AND `token`=?"

	if rtToken, ok = preProcessCliToken(rtToken); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	if err := v.DB.QueryRowContext(ctx, queryCmd, rtToken).Scan(&ok); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)SetRoot(token string, value bool)(err error){
	const updateCmd = "UPDATE tokens SET" +
		" `root`=?" +
		" WHERE `token`=?"

	var ok bool
	if token, ok = preProcessCliToken(token); !ok {
		err = TokenNotExistsErr
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	tx, err := v.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err = execTx(tx, updateCmd, value, token); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)CreateServer(id string)(err error){
	const insertCmd = "INSERT INTO servers (`id`)" +
		" VALUES (?)"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	tx, err := v.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err = execTx(tx, insertCmd, id); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)RemoveServer(id string)(err error){
	const deleteCmd = "DELETE FROM servers" +
		" WHERE `id`=?"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	tx, err := v.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err = execTx(tx, deleteCmd, id); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)ListServers(token string)(servers []string, err error){
	const query1Cmd = "SELECT `root` FROM tokens" +
		" WHERE (`expiration` IS NULL OR CONVERT_TZ(`expiration`,@@session.time_zone,'+00:00')>=NOW())" +
		" AND `token`=?"
	const query2Cmd = "SELECT `id` FROM servers"
	const query3Cmd = "SELECT `server` FROM token_ops" +
		" WHERE `token`=?"

	var ok bool
	if token, ok = preProcessCliToken(token); !ok {
		err = PermDeniedErr
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	var root bool
	if err = v.DB.QueryRowContext(ctx, query1Cmd, token).Scan(&root); err != nil {
		return
	}
	var args []any
	var queryCmd2 string
	if root {
		queryCmd2 = query2Cmd
	}else{
		queryCmd2 = query3Cmd
		args = []any{token}
	}

	var rows *sql.Rows
	if rows, err = v.QueryContext(ctx, queryCmd2, args...); err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var server string
		if err = rows.Scan(&server); err != nil {
			return
		}
		servers = append(servers, server)
	}
	if err = rows.Err(); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)CheckPerm(token string, server string)(ok bool){
	const query1Cmd = "SELECT `root` FROM tokens" +
		" WHERE (`expiration` IS NULL OR CONVERT_TZ(`expiration`,@@session.time_zone,'+00:00')>=NOW())" +
		" AND `token`=?"
	const query2Cmd = "SELECT 1 FROM token_ops" +
		" WHERE `token`=? AND `server`=?"

	if token, ok = preProcessCliToken(token); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	// check if it's a root token
	if err := v.DB.QueryRowContext(ctx, query1Cmd, token).Scan(&ok); err != nil {
		return
	}
	if ok {
		return
	}

	if err := v.DB.QueryRowContext(ctx, query2Cmd, token, server).Scan(&ok); err != nil {
		return
	}
	return
}

func (v *MySQLAPI)SetPerm(token string, server string, value bool)(err error){
	const insertCmd = "INSERT IGNORE INTO token_ops (`token`, `server`)" +
		" VALUES (?, ?)"
	const deleteCmd = "DELETE FROM token_ops" +
		" WHERE `token`=? AND `server`=?"

	var ok bool
	if token, ok = preProcessCliToken(token); !ok {
		err = TokenNotExistsErr
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	tx, err := v.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	var cmd string
	if value {
		cmd = insertCmd
	}else{
		cmd = deleteCmd
	}
	if _, err = execTx(tx, cmd, token, server); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}
	return
}

func generateToken()(token string, err error){
	var buf [tokenLen * 3 / 4]byte
	if _, err = crand.Read(buf[:]); err != nil {
		return
	}
	token = base64.RawURLEncoding.EncodeToString(buf[:])
	token = token[:tokenLen]
	return
}

func execTx(tx *sql.Tx, cmd string, args ...any)(res sql.Result, err error){
	loger.Debugf("Exec sql cmd: %s\n  args: %v", cmd, args)
	for {
		if res, err = tx.Exec(cmd, args...); err != nil {
			if e, ok := err.(*mysql.MySQLError); ok {
				switch e.Number {
				case mysqlDeadLockCode:
					continue
				}
			}
		}
		return
	}
}

