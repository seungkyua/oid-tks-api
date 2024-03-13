package audit

import (
	"bytes"
	"encoding/json"
	"fmt"

	internalApi "github.com/openinfradev/tks-api/internal/delivery/api"
	"github.com/openinfradev/tks-api/pkg/domain"
	"github.com/openinfradev/tks-api/pkg/httpErrors"
	"github.com/openinfradev/tks-api/pkg/log"
)

type fnAudit = func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string)

var auditMap = map[internalApi.Endpoint]fnAudit{
	internalApi.CreateStack: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreateStackRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		if isSuccess(statusCode) {
			return fmt.Sprintf("스택 [%s]을 생성하였습니다.", input.Name), ""
		} else {
			return fmt.Sprintf("스택 [%s]을 생성하는데 실패하였습니다.", input.Name), errorText(out)
		}
	}, internalApi.CreateProject: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreateProjectRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		if isSuccess(statusCode) {
			return fmt.Sprintf("프로젝트 [%s]를 생성하였습니다.", input.Name), ""
		} else {
			return fmt.Sprintf("프로젝트 [%s]을 생성하는데 실패하였습니다.", input.Name), errorText(out)
		}
	}, internalApi.CreateCloudAccount: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreateCloudAccountRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		if isSuccess(statusCode) {
			return fmt.Sprintf("클라우드 어카운트 [%s]를 생성하였습니다.", input.Name), ""
		} else {
			return fmt.Sprintf("클라우드 어카운트 [%s]을 생성하는데 실패하였습니다.", input.Name), errorText(out)
		}
	}, internalApi.DeleteCloudAccount: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		if isSuccess(statusCode) {
			output := domain.DeleteCloudAccountResponse{}
			if err := json.Unmarshal(in, &output); err != nil {
				log.Error(err)
			}
			return fmt.Sprintf("클라우드어카운트 [ID:%s]를 삭제하였습니다.", output.ID), ""
		} else {
			return "클라우드어카운트 [%s]를 삭제하는데 실패하였습니다. ", errorText(out)
		}
	}, internalApi.CreateUser: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreateUserRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		if isSuccess(statusCode) {
			return fmt.Sprintf("사용자 [%s]를 생성하였습니다.", input.Name), ""
		} else {
			return fmt.Sprintf("사용자 [%s]을 생성하는데 실패하였습니다.", input.Name), errorText(out)
		}
	}, internalApi.CreateOrganization: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreateOrganizationRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		if isSuccess(statusCode) {
			return fmt.Sprintf("조직 [%s]를 생성하였습니다.", input.Name), ""
		} else {
			return fmt.Sprintf("조직 [%s]을 생성하는데 실패하였습니다.", input.Name), errorText(out)
		}
	}, internalApi.DeleteOrganization: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		if isSuccess(statusCode) {
			output := domain.DeleteOrganizationResponse{}
			if err := json.Unmarshal(in, &output); err != nil {
				log.Error(err)
			}
			return fmt.Sprintf("조직 [ID:%s]를 삭제하였습니다.", output.ID), ""
		} else {
			return "조직 [%s]를 삭제하는데 실패하였습니다. ", errorText(out)
		}
	}, internalApi.CreateAppServeApp: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreateAppServeAppRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		if isSuccess(statusCode) {
			return fmt.Sprintf("앱서빙 [%s]를 생성하였습니다.", input.Name), ""
		} else {
			return fmt.Sprintf("앱서빙 [%s]을 생성하는데 실패하였습니다.", input.Name), errorText(out)
		}
	}, internalApi.Admin_CreateStackTemplate: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreateStackTemplateRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		log.Info(input)
		if isSuccess(statusCode) {
			return fmt.Sprintf("스택 템플릿 [%s]를 생성하였습니다.", input.Name), ""
		} else {
			return fmt.Sprintf("스택 템플릿 [%s]을 생성하는데 실패하였습니다.", input.Name), errorText(out)
		}
	}, internalApi.Admin_CreateUser: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreateUserRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		if isSuccess(statusCode) {
			return fmt.Sprintf("어드민 [%s]를 생성하였습니다.", input.Name), ""
		} else {
			return fmt.Sprintf("어드민 [%s]을 생성하는데 실패하였습니다.", input.Name), errorText(out)
		}
	}, internalApi.CreatePolicyTemplate: func(out *bytes.Buffer, in []byte, statusCode int) (message string, description string) {
		input := domain.CreatePolicyTemplateRequest{}
		if err := json.Unmarshal(in, &input); err != nil {
			log.Error(err)
		}
		if isSuccess(statusCode) {
			return fmt.Sprintf("폴리시템플릿 [%s]를 생성하였습니다.", input.TemplateName), ""
		} else {
			return fmt.Sprintf("폴리시템플릿 [%s]을 생성하는데 실패하였습니다.", input.TemplateName), errorText(out)
		}
	},
}

func errorText(out *bytes.Buffer) string {
	var e httpErrors.RestError
	if err := json.NewDecoder(out).Decode(&e); err != nil {
		log.Error(err)
		return ""
	}
	return e.Text()
}

func isSuccess(statusCode int) bool {
	if statusCode >= 200 && statusCode < 300 {
		return true
	}
	return false
}
