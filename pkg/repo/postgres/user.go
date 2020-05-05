package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/xssnick/wint/pkg/repo"
)

func (r *Repo) GetUserIDByPhone(ctx context.Context, phone uint64) (uint64, error) {
	var id uint64

	err := r.db.GetContext(ctx, &id, "SELECT id FROM users WHERE phone=$1", phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, repo.ErrNotFound
		}

		return 0, err
	}

	return id, nil
}

func (r *Repo) GetUserProfile(ctx context.Context, id uint64) (*repo.UserProfile, error) {
	p := struct {
		Phone       uint64     `db:"phone"`
		Name        string     `db:"name"`
		City        string     `db:"city"`
		Country     string     `db:"country"`
		Description string     `db:"description"`
		Mode        int        `db:"mode"`
		RegFinished bool       `db:"reg_finished"`
		Sex         int        `db:"sex"`
		CreatedAt   time.Time  `db:"created_at"`
		Birth       *time.Time `db:"birth"`
	}{}

	err := r.db.GetContext(ctx, &p, "SELECT phone,name,city,country,created_at,description,reg_finished,birth,sex,mode FROM users WHERE id=$1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrNotFound
		}

		return nil, err
	}

	return &repo.UserProfile{
		ID:          id,
		Phone:       p.Phone,
		Name:        p.Name,
		Mode:        p.Mode,
		City:        p.City,
		Country:     p.Country,
		Description: p.Description,
		RegFinished: p.RegFinished,
		Birth:       p.Birth,
		Sex:         p.Sex,
		CreatedAt:   p.CreatedAt,
	}, nil
}

func (r *Repo) CreateUser(ctx context.Context, phone uint64) (uint64, error) {
	var id uint64

	err := r.db.GetContext(ctx, &id, "INSERT INTO users (phone,created_at) VALUES ($1,$2) RETURNING id", phone, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repo) EditProfile(ctx context.Context, id uint64, data repo.UserEdit) error {
	mode := 0
	switch data.Sex {
	case 1:
		mode = 2
	case 2:
		mode = 1
	}

	editor := struct {
		ID          uint64     `db:"id"`
		Name        string     `db:"name"`
		City        string     `db:"city"`
		Country     string     `db:"country"`
		Description string     `db:"description"`
		RegFinished bool       `db:"reg_finished"`
		Sex         int        `db:"sex"`
		Mode        int        `db:"mode"`
		Birth       *time.Time `db:"birth"`
	}{
		ID:          id,
		Name:        data.Name,
		RegFinished: true,
		City:        data.City,
		Country:     data.Country,
		Description: data.Description,
		Sex:         data.Sex,
		Mode:        mode,
		Birth:       &data.Birth,
	}

	_, err := r.db.NamedExecContext(ctx, "UPDATE users SET mode=:mode,name=:name,reg_finished=:reg_finished,city=:city,country=:country,description=:description,sex=:sex,birth=:birth WHERE id=:id", editor)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) GetUsersNames(ctx context.Context, uids []uint64) ([]repo.UserNameID, error) {
	if len(uids) == 0 {
		return []repo.UserNameID{}, nil
	}

	ids := "("
	for i, id := range uids {
		if i != 0 {
			ids += ","
		}

		ids += strconv.FormatUint(id, 10)
	}
	ids += ")"

	var pls []struct {
		ID   uint64 `db:"id"`
		Name string `db:"name"`
	}

	err := r.db.SelectContext(ctx, &pls, `SELECT id, name FROM users WHERE id IN `+ids)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	res := make([]repo.UserNameID, len(pls))
	for i, v := range pls {
		res[i].ID = v.ID
		res[i].Name = v.Name
	}

	return res, nil
}
