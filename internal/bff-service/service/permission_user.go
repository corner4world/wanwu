package service

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"regexp"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	iam_service "github.com/UnicomAI/wanwu/api/proto/iam-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

// --- user excel import constants ---
const (
	ExcelHeaderUserName      = "用户名"
	ExcelHeaderPassword      = "密码"
	ExcelHeaderCompany       = "单位"
	ExcelHeaderPhone         = "电话"
	ExcelHeaderRole          = "角色"
	ExcelHeaderRemark        = "备注"
	MaxBatchCreateUsersLimit = 500
)

var requiredUserExcelHeaders = []string{
	ExcelHeaderUserName,
	ExcelHeaderPassword,
	ExcelHeaderCompany,
	ExcelHeaderPhone,
	ExcelHeaderRole,
	ExcelHeaderRemark,
}

func CreateUser(ctx *gin.Context, creatorID, orgID string, userCreate *request.UserCreate) (*response.UserID, error) {
	password, err := decryptPD(userCreate.Password)
	if err != nil {
		return nil, fmt.Errorf("decrypt password err: %v", err)
	}
	resp, err := iam.CreateUser(ctx.Request.Context(), &iam_service.CreateUserReq{
		CreatorId: creatorID,
		OrgId:     orgID,
		UserName:  userCreate.Username,
		NickName:  userCreate.Nickname,
		Gender:    userCreate.Gender,
		Phone:     userCreate.Phone,
		Company:   userCreate.Company,
		Remark:    userCreate.Remark,
		Password:  password,
		RoleIds:   userCreate.RoleIDs,
	})
	if err != nil {
		return nil, err
	}
	return &response.UserID{UserID: resp.Id}, nil
}

func ChangeUser(ctx *gin.Context, orgID string, userUpdate *request.UserUpdate) error {
	_, err := iam.UpdateUser(ctx.Request.Context(), &iam_service.UpdateUserReq{
		UserId:   userUpdate.UserID,
		OrgId:    orgID,
		NickName: userUpdate.Nickname,
		Gender:   userUpdate.Gender,
		Phone:    userUpdate.Phone,
		Company:  userUpdate.Company,
		Remark:   userUpdate.Remark,
		RoleIds:  userUpdate.RoleIDs,
	})
	return err
}

func DeleteUser(ctx *gin.Context, userID string) error {
	_, err := iam.DeleteUser(ctx.Request.Context(), &iam_service.DeleteUserReq{
		UserId: userID,
	})
	return err
}

func GetUserInfo(ctx *gin.Context, userID, orgID string) (*response.UserInfo, error) {
	resp, err := iam.GetUserInfo(ctx.Request.Context(), &iam_service.GetUserInfoReq{
		UserId: userID,
		OrgId:  orgID,
	})
	if err != nil {
		return nil, err
	}
	return toUserInfo(ctx, resp), nil
}

func GetUserList(ctx *gin.Context, orgID, name string, pageNo, pageSize int32) (*response.PageResult, error) {
	resp, err := iam.GetUserList(ctx.Request.Context(), &iam_service.GetUserListReq{
		OrgId:    orgID,
		UserName: name,
		PageNo:   pageNo,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	var users []*response.UserInfo
	for _, user := range resp.Users {
		users = append(users, toUserInfo(ctx, user))
	}
	return &response.PageResult{
		List:     users,
		Total:    resp.Total,
		PageNo:   int(pageNo),
		PageSize: int(pageSize),
	}, nil
}

func ChangeUserStatus(ctx *gin.Context, userID, orgID string, status bool) error {
	_, err := iam.ChangeUserStatus(ctx.Request.Context(), &iam_service.ChangeUserStatusReq{
		UserId: userID,
		OrgId:  orgID,
		Status: status,
	})
	return err
}

func ChangeUserPassword(ctx *gin.Context, userID, oldPwd, newPwd string) error {
	oldPassword, err := decryptPD(oldPwd)
	if err != nil {
		return fmt.Errorf("decrypt password err: %v", err)
	}
	newPassword, err := decryptPD(newPwd)
	if err != nil {
		return fmt.Errorf("decrypt password err: %v", err)
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	_, err = iam.UpdateUserPassword(ctx.Request.Context(), &iam_service.UpdateUserPasswordReq{
		UserId:      userID,
		OldPassword: oldPassword,
		NewPassword: newPassword,
	})
	return err
}

func AdminChangeUserPassword(ctx *gin.Context, userID, pwd string) error {
	password, err := decryptPD(pwd)
	if err != nil {
		return fmt.Errorf("decrypt password err: %v", err)
	}
	_, err = iam.ResetUserPassword(ctx.Request.Context(), &iam_service.ResetUserPasswordReq{
		UserId:   userID,
		Password: password,
	})
	return err
}

func GetOrgUserNotSelect(ctx *gin.Context, orgID, name string) (*response.Select, error) {
	users, err := iam.GetUserSelectNotInOrg(ctx.Request.Context(), &iam_service.GetUserSelectNotInOrgReq{
		OrgId:    orgID,
		UserName: name,
	})
	if err != nil {
		return nil, err
	}
	return &response.Select{Select: toIDNames(users.Selects)}, nil
}

func GetRoleSelect(ctx *gin.Context, orgID string) (*response.Select, error) {
	roles, err := iam.GetRoleSelect(ctx.Request.Context(), &iam_service.GetRoleSelectReq{
		OrgId: orgID,
	})
	if err != nil {
		return nil, err
	}
	return &response.Select{Select: toRoleIDNames(ctx, roles.Roles)}, nil
}

func AddOrgUser(ctx *gin.Context, orgID, userID, roleID string) error {
	_, err := iam.AddOrgUser(ctx.Request.Context(), &iam_service.AddOrgUserReq{
		OrgId:  orgID,
		UserId: userID,
		RoleId: roleID,
	})
	return err
}

func RemoveOrgUser(ctx *gin.Context, orgID, userID string) error {
	_, err := iam.RemoveOrgUser(ctx.Request.Context(), &iam_service.RemoveOrgUserReq{
		OrgId:  orgID,
		UserId: userID,
	})
	return err
}

func UpdateUserAvatar(ctx *gin.Context, userID, key string) error {
	_, err := iam.UpdateUserAvatar(ctx.Request.Context(), &iam_service.UpdateUserAvatarReq{
		UserId:     userID,
		AvatarPath: key,
	})
	return err
}

func CreateUserByFile(ctx *gin.Context, creatorID, orgID string) error {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("get file err: %v", err))
	}
	file, err := fileHeader.Open()
	if err != nil {
		return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("open file err: %v", err))
	}
	defer func() { _ = file.Close() }()
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("read file err: %v", err))
	}

	users, err := parseUserExcel(fileBytes)
	if err != nil {
		return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("parse excel err: %v", err))
	}

	if len(users) > MaxBatchCreateUsersLimit {
		return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("批量创建用户条数不能超过%d条", MaxBatchCreateUsersLimit))
	}

	for _, user := range users {
		if err := validateUsername(user.UserName); err != nil {
			return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("username %s: %v", user.UserName, err))
		}
		if err := validatePassword(user.Password); err != nil {
			return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("username %s: %v", user.UserName, err))
		}
		if user.Phone != "" {
			if err := validatePhone(user.Phone); err != nil {
				return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("phone %s: %v", user.Phone, err))
			}
		}
		if user.Company == "" {
			return grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_user_batch_import_file", fmt.Sprintf("username %s: company is empty", user.UserName))
		}
	}

	_, err = iam.CreateUsers(ctx.Request.Context(), &iam_service.CreateUsersReq{
		CreatorId: creatorID,
		OrgId:     orgID,
		Users:     users,
	})
	return err
}

func parseUserExcel(fileData []byte) ([]*iam_service.CreateUsersInfo, error) {
	wb, err := util.OpenWorkbookFromBytes(fileData)
	if err != nil {
		return nil, err
	}
	defer func() { _ = wb.Close() }()

	sheets, err := wb.GetSheets()
	if err != nil {
		return nil, fmt.Errorf("excel has no sheets")
	}
	if len(sheets) == 0 {
		return nil, fmt.Errorf("excel has no sheets")
	}
	rows, err := wb.GetRows("")
	if err != nil {
		return nil, fmt.Errorf("invalid excel data")
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("excel has no data rows")
	}
	headerRow := rows[0]
	headerSet := make(map[string]bool)
	for _, h := range headerRow {
		headerSet[h] = true
	}
	for _, required := range requiredUserExcelHeaders {
		if !headerSet[required] {
			return nil, fmt.Errorf("excel header invalid: missing %s", required)
		}
	}

	records, err := wb.ReadWithHeaderMapping(util.ReadWithHeaderMappingOptions{
		Sheet:     "",
		HeaderRow: 0,
		HeaderMapping: map[string]string{
			ExcelHeaderUserName: "userName",
			ExcelHeaderPassword: "password",
			ExcelHeaderCompany:  "company",
			ExcelHeaderPhone:    "phone",
			ExcelHeaderRole:     "roleName",
			ExcelHeaderRemark:   "remark",
		},
	})
	if err != nil {
		return nil, err
	}

	var users []*iam_service.CreateUsersInfo
	for _, record := range records {
		if record["userName"] == "" {
			continue
		}
		users = append(users, &iam_service.CreateUsersInfo{
			UserName: record["userName"],
			Password: record["password"],
			Company:  record["company"],
			Phone:    record["phone"],
			RoleName: record["roleName"],
			Remark:   record["remark"],
		})
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("no valid user data")
	}
	return users, nil
}

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9\x{4e00}-\x{9fa5}_().]+$`)
	phoneRegex    = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

func validateUsername(username string) error {
	if len(username) < 2 || len(username) > 20 {
		return fmt.Errorf("用户名长度需为2-20个字符")
	}
	if username[0] == '_' {
		return fmt.Errorf("用户名不能以下划线开头")
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("用户名只能包含中英文、数字、下划线、括号")
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("请输入密码")
	}
	if len(password) < 8 || len(password) > 20 {
		return fmt.Errorf("密码长度需为8-20个字符")
	}
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)
	if !hasLetter || !hasNumber || !hasSpecial {
		return fmt.Errorf("密码需包含字母、数字、特殊字符")
	}
	return nil
}

func validatePhone(phone string) error {
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("电话号码格式不正确")
	}
	return nil
}

// --- internal ---

func toUserInfo(ctx *gin.Context, user *iam_service.UserInfo) *response.UserInfo {
	ret := &response.UserInfo{
		UserID:    user.UserId,
		Username:  user.UserName,
		Nickname:  user.NickName,
		Phone:     user.Phone,
		Email:     user.Email,
		Gender:    user.Gender,
		Remark:    user.Remark,
		Company:   user.Company,
		CreatedAt: util.Time2Str(user.CreatedAt),
		Creator:   toIDName(user.Creator),
		Status:    user.Status,
		Language:  getLanguageByCode(user.Language),
		Avatar:    cacheUserAvatar(ctx, user.AvatarPath),
	}
	for _, userOrg := range user.Orgs {
		ret.Orgs = append(ret.Orgs, toOrgRole(ctx, userOrg))
	}
	return ret
}

func toOrgRole(ctx *gin.Context, userOrg *iam_service.UserOrg) response.OrgRole {
	return response.OrgRole{
		Org:   toOrgIDName(ctx, userOrg.Org),
		Roles: toRoleIDNames(ctx, userOrg.Roles),
	}
}

// 解密password
func decryptPD(encryptStr string) (string, error) {
	var (
		err                      error
		urlUnescape              string
		base64Decode, decryptAes []byte
	)
	if encryptStr == "" {
		return "", nil
	}

	if urlUnescape, err = url.QueryUnescape(encryptStr); nil != err {
		return "", err
	}

	if base64Decode, err = base64.StdEncoding.DecodeString(urlUnescape); nil != err {
		return "", err
	}

	iv := []byte(config.Cfg().Decrypt.IV)
	key := []byte(config.Cfg().Decrypt.Key)
	if decryptAes, err = util.DecryptAES(base64Decode, key, iv); nil != err {
		return "", err
	}

	return string(decryptAes), nil
}
