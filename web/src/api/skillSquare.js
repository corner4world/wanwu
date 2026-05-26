// @description: skill广场相关接口
import request from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';

// 获取skill广场列表-内置
export const getBuiltinSquareSkillList = data => {
  return request({
    url: `${USER_API}/square/skill/builtin/list`,
    method: 'get',
    params: data,
  });
};

// 发送skill广场到资源库
export const sendSquareSkillToResource = data => {
  return request({
    url: `${USER_API}/square/skill/share`,
    method: 'post',
    data,
  });
};
// 获取skill广场内置详情
export const getSquareSkillDetail = data => {
  return request({
    url: `${USER_API}/square/skill/builtin/detail`,
    method: 'get',
    params: data,
  });
};

// 下载skill广场skill
export const downloadSquareSkill = params => {
  return request({
    url: `${USER_API}/square/skill/download`,
    method: 'get',
    params,
    responseType: 'blob',
  });
};

// 获取广场共享skill列表
export const getSharedSquareSkillList = params => {
  return request({
    url: `${USER_API}/square/skill/share/list`,
    method: 'get',
    params,
  });
};

// 添加共享skill到资源库
export const addSharedSkillToResource = data => {
  return request({
    url: `${USER_API}/square/skill/share`,
    method: 'post',
    data,
  });
};

// 下载共享skill
export const downloadSharedSquareSkill = params => {
  return request({
    url: `${USER_API}/square/skill/share/download`,
    method: 'get',
    params,
    responseType: 'blob',
  });
};

// 获取共享skill详情
export const getSharedSquareSkillDetail = params => {
  return request({
    url: `${USER_API}/square/skill/share/detail`,
    method: 'get',
    params,
  });
};

// 获取我发布的 Skill 列表
export const getCreatedSquareSkillList = params => {
  return request({
    url: `${USER_API}/square/skill/created/list`,
    method: 'get',
    params,
  });
};

// 获取我发布的 Skill 详情
export const getCreatedSquareSkillDetail = params => {
  return request({
    url: `${USER_API}/square/skill/created/detail`,
    method: 'get',
    params,
  });
};

// 获取我发布的 Skill 版本列表
export const getCreatedSkillVersionList = params => {
  return request({
    url: `${USER_API}/square/skill/created/version/list`,
    method: 'get',
    params,
  });
};

// 获取共享 Skill 版本列表
export const getSharedSkillVersionList = params => {
  return request({
    url: `${USER_API}/square/skill/share/version/list`,
    method: 'get',
    params,
  });
};
