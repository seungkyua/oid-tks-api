package usecase

import (
	"context"
	"fmt"
	"github.com/Nerzal/gocloak/v13"
	"github.com/google/uuid"
	"github.com/openinfradev/tks-api/internal/auth/request"
	"github.com/openinfradev/tks-api/internal/helper"
	"github.com/openinfradev/tks-api/internal/keycloak"
	"github.com/openinfradev/tks-api/internal/repository"
	"github.com/openinfradev/tks-api/pkg/domain"
	"github.com/openinfradev/tks-api/pkg/httpErrors"
	"github.com/pkg/errors"
	"net/http"
)

type IUserUsecase interface {
	CreateAdmin(organizationId string) (*domain.User, error)
	DeleteAdmin(organizationId string) error
	DeleteAll(ctx context.Context, organizationId string) error
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	List(ctx context.Context) (*[]domain.User, error)
	GetByAccountId(ctx context.Context, accountId string) (*domain.User, error)
	UpdateByAccountId(ctx context.Context, accountId string, user *domain.User) (*domain.User, error)
	UpdatePasswordByAccountId(ctx context.Context, accountId string, password string) error
	DeleteByAccountId(ctx context.Context, accountId string) error
}

type UserUsecase struct {
	repo repository.IUserRepository
	kc   keycloak.IKeycloak
}

func (u *UserUsecase) DeleteAll(ctx context.Context, organizationId string) error {
	users, err := u.repo.List(u.repo.OrganizationFilter(organizationId))
	if err != nil {
		return err
	}
	token, ok := request.TokenFrom(ctx)
	if ok == false {
		return httpErrors.NewInternalServerError(fmt.Errorf("token in the context is empty"))
	}
	for _, user := range *users {
		// Delete user in keycloak

		err = u.kc.DeleteUser(organizationId, user.AccountId, token)
		if err != nil {
			if _, statusCode := httpErrors.ErrorResponse(err); statusCode == http.StatusNotFound {
				continue
			}
			return err
		}

		uuid, err := uuid.Parse(user.ID)
		if err != nil {
			return err
		}
		err = u.repo.DeleteWithUuid(uuid)
		if err != nil {
			if _, statusCode := httpErrors.ErrorResponse(err); statusCode == http.StatusNotFound {
				continue
			}
			return err
		}
	}

	return nil
}

func (u *UserUsecase) DeleteAdmin(organizationId string) error {
	token, err := u.kc.LoginAdmin()
	if err != nil {
		return errors.Wrap(err, "login admin failed")
	}

	user, err := u.kc.GetUser(organizationId, "admin", token)
	if err != nil {
		return errors.Wrap(err, "get user failed")
	}

	err = u.kc.DeleteUser(organizationId, "admin", token)
	if err != nil {
		return errors.Wrap(err, "delete user failed")
	}

	userUuid, err := uuid.Parse(*user.ID)
	if err != nil {
		return errors.Wrap(err, "parse user id failed")
	}

	err = u.repo.DeleteWithUuid(userUuid)
	if err != nil {
		return errors.Wrap(err, "delete user failed")
	}

	return nil
}

func (u *UserUsecase) CreateAdmin(orgainzationId string) (*domain.User, error) {
	token, err := u.kc.LoginAdmin()
	if err != nil {
		return nil, errors.Wrap(err, "login admin failed")
	}
	user := domain.User{
		AccountId: "admin",
		Password:  "admin",
		Role: domain.Role{
			Name: "admin",
		},
		Organization: domain.Organization{
			ID: orgainzationId,
		},
		Name: "admin",
	}

	// Create user in keycloak
	groups := []string{fmt.Sprintf("%s@%s", user.Role.Name, orgainzationId)}
	err = u.kc.CreateUser(orgainzationId, &gocloak.User{
		Username: gocloak.StringP(user.AccountId),
		Credentials: &[]gocloak.CredentialRepresentation{
			{
				Type:      gocloak.StringP("password"),
				Value:     gocloak.StringP(user.Password),
				Temporary: gocloak.BoolP(false),
			},
		},
		Groups: &groups,
	}, token)
	if err != nil {
		return nil, errors.Wrap(err, "creating user in keycloak failed")
	}
	keycloakUser, err := u.kc.GetUser(user.Organization.ID, user.AccountId, token)
	if err != nil {
		return nil, errors.Wrap(err, "getting user from keycloak failed")
	}

	userUuid, err := uuid.Parse(*keycloakUser.ID)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := helper.HashPassword(user.Password)
	if err != nil {
		return nil, err
	}

	roles, err := u.repo.FetchRoles()
	if err != nil {
		return nil, err
	}
	for _, role := range *roles {
		if role.Name == user.Role.Name {
			user.Role.ID = role.ID
		}
	}
	roleUuid, err := uuid.Parse(user.Role.ID)
	if err != nil {
		return nil, err
	}
	resUser, err := u.repo.CreateWithUuid(userUuid, user.AccountId, user.Name, hashedPassword, user.Email,
		user.Department, user.Description, user.Organization.ID, roleUuid)
	if err != nil {
		return nil, err
	}

	//err = u.repo.AssignRole(user.AccountId, user.Organization.ID, user.Role.Name)
	//if err != nil {
	//	return nil, err
	//}

	return &resUser, nil
}

func (u *UserUsecase) UpdatePasswordByAccountId(ctx context.Context, accountId string, newPassword string) error {

	token, ok := request.TokenFrom(ctx)
	if ok == false {
		return fmt.Errorf("token in the context is empty")
	}

	userInfo, ok := request.UserFrom(ctx)
	if ok == false {
		return fmt.Errorf("user in the context is empty")
	}

	originUser, err := u.kc.GetUser(userInfo.GetOrganizationId(), accountId, token)
	if err != nil {
		return errors.Wrap(err, "getting user from keycloak failed")
	}

	originUser.Credentials = &[]gocloak.CredentialRepresentation{
		{
			Type:      gocloak.StringP("password"),
			Value:     gocloak.StringP(newPassword),
			Temporary: gocloak.BoolP(false),
		},
	}

	err = u.kc.UpdateUser(userInfo.GetOrganizationId(), originUser, token)
	if err != nil {
		return errors.Wrap(err, "updating user in keycloak failed")
	}

	// update password in DB

	user, err := u.repo.Get(accountId, userInfo.GetOrganizationId())
	if err != nil {
		return errors.Wrap(err, "getting user from repository failed")
	}
	uuid, err := uuid.Parse(user.ID)
	if err != nil {
		return errors.Wrap(err, "parsing uuid failed")
	}
	hashedPassword, err := helper.HashPassword(newPassword)
	if err != nil {
		return errors.Wrap(err, "hashing password failed")
	}

	_, err = u.repo.UpdateWithUuid(uuid, user.AccountId, user.Name, hashedPassword, user.Email,
		user.Department, user.Description)
	if err != nil {
		return errors.Wrap(err, "updating user in repository failed")
	}

	return nil
}

func (u *UserUsecase) List(ctx context.Context) (*[]domain.User, error) {
	userInfo, ok := request.UserFrom(ctx)
	if ok == false {
		return nil, fmt.Errorf("user in the context is empty")
	}

	users, err := u.repo.List(u.repo.OrganizationFilter(userInfo.GetOrganizationId()))
	if err != nil {
		return nil, errors.Wrap(err, "getting users from repository failed")
	}

	return users, nil
}

func (u *UserUsecase) GetByAccountId(ctx context.Context, accountId string) (*domain.User, error) {
	userInfo, ok := request.UserFrom(ctx)
	if ok == false {
		return nil, fmt.Errorf("user in the context is empty")
	}

	users, err := u.repo.List(u.repo.OrganizationFilter(userInfo.GetOrganizationId()),
		u.repo.AccountIdFilter(accountId))
	if err != nil {
		return nil, err
	}

	return &(*users)[0], nil
}

func (u *UserUsecase) UpdateByAccountId(ctx context.Context, accountId string, user *domain.User) (*domain.User, error) {
	userInfo, ok := request.UserFrom(ctx)
	if ok == false {
		return nil, fmt.Errorf("user in the context is empty")
	}

	users, err := u.repo.List(u.repo.OrganizationFilter(userInfo.GetOrganizationId()),
		u.repo.AccountIdFilter(accountId))
	if err != nil {
		return nil, errors.Wrap(err, "getting users from repository failed")
	}
	if len(*users) == 0 {
		return nil, fmt.Errorf("user not found")
	} else if len(*users) > 1 {
		return nil, fmt.Errorf("multiple users found")
	}

	uuid, err := uuid.Parse((*users)[0].ID)
	if err != nil {
		return nil, err
	}

	originPassword := (*users)[0].Password

	*user, err = u.repo.UpdateWithUuid(uuid, user.AccountId, user.Name, originPassword, user.Email,
		user.Department, user.Description)
	if err != nil {
		return nil, errors.Wrap(err, "updating user in repository failed")
	}

	return user, nil
}

func (u *UserUsecase) DeleteByAccountId(ctx context.Context, accountId string) error {
	userInfo, ok := request.UserFrom(ctx)
	if ok == false {
		return fmt.Errorf("user in the context is empty")
	}

	user, err := u.repo.Get(accountId, userInfo.GetOrganizationId())
	if err != nil {
		return errors.Wrap(err, "getting users from repository failed")
	}

	uuid, err := uuid.Parse(user.ID)
	if err != nil {
		return err
	}
	err = u.repo.DeleteWithUuid(uuid)
	if err != nil {
		return errors.Wrap(err, "deleting user in repository failed")
	}

	// Delete user in keycloak
	token, ok := request.TokenFrom(ctx)
	if ok == false {
		return fmt.Errorf("token in the context is empty")
	}
	err = u.kc.DeleteUser(userInfo.GetOrganizationId(), accountId, token)
	if err != nil {
		return errors.Wrap(err, "deleting user in keycloak failed")
	}

	return nil
}

func (u *UserUsecase) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	// Validation check

	token, ok := request.TokenFrom(ctx)
	if ok == false {
		return nil, fmt.Errorf("token in the context is empty")
	}

	// Create user in keycloak
	groups := []string{fmt.Sprintf("%s@%s", user.Role.Name, user.Organization.ID)}
	err := u.kc.CreateUser(user.Organization.ID, &gocloak.User{
		Username: gocloak.StringP(user.AccountId),
		Credentials: &[]gocloak.CredentialRepresentation{
			{
				Type:      gocloak.StringP("password"),
				Value:     gocloak.StringP(user.Password),
				Temporary: gocloak.BoolP(false),
			},
		},
		Groups: &groups,
	}, token)
	if err != nil {
		return nil, errors.Wrap(err, "creating user in keycloak failed")
	}
	keycloakUser, err := u.kc.GetUser(user.Organization.ID, user.AccountId, token)
	if err != nil {
		return nil, errors.Wrap(err, "getting user from keycloak failed")
	}

	userUuid, err := uuid.Parse(*keycloakUser.ID)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := helper.HashPassword(user.Password)
	if err != nil {
		return nil, err
	}

	roles, err := u.repo.FetchRoles()
	if err != nil {
		return nil, err
	}
	for _, role := range *roles {
		if role.Name == user.Role.Name {
			user.Role.ID = role.ID
		}
	}
	roleUuid, err := uuid.Parse(user.Role.ID)
	if err != nil {
		return nil, err
	}

	resUser, err := u.repo.CreateWithUuid(userUuid, user.AccountId, user.Name, hashedPassword, user.Email,
		user.Department, user.Description, user.Organization.ID, roleUuid)
	if err != nil {
		return nil, err
	}

	return &resUser, nil
}

func NewUserUsecase(r repository.IUserRepository, kc keycloak.IKeycloak) IUserUsecase {
	return &UserUsecase{
		repo: r,
		kc:   kc,
	}
}
