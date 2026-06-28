package graph

import (
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	orgUC "github.com/open-git/backend/internal/usecase/org"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
	userUC "github.com/open-git/backend/internal/usecase/user"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	UserRepo       domainrepo.IUserRepository
	LabelRepo      domainrepo.ILabelRepository
	MilestoneRepo  domainrepo.IMilestoneRepository
	RepositoryRepo domainrepo.IRepositoryRepository

	GetCurrentUserUC *userUC.GetCurrentUserUsecase
	GetUserByLoginUC *userUC.GetUserByLoginUsecase
	GetRepositoryUC  *repoUC.GetRepositoryUsecase
	GetOrgUC         *orgUC.GetOrgUsecase
	CreateIssueUC    *issueusecase.CreateIssueUsecase
	UpdateIssueUC    *issueusecase.UpdateIssueUsecase
	CreateCommentUC  *issueusecase.CreateCommentUsecase
	ListIssuesUC     *issueusecase.ListIssuesUsecase
	CreatePRUC       *prusecase.CreatePRUsecase
	MergePRUC        *prusecase.MergePRUsecase
}
