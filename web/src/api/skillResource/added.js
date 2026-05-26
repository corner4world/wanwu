// @description: 我添加的skill相关接口
import request from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';

// 获取我添加的skill列表
export const getAcquiredSkillList = data => {
  return request({
    url: `${USER_API}/agent/skill/acquired/list`,
    method: 'get',
    params: data,
  });
};

// 下载我添加的skill
export const downloadAcquiredSkill = params => {
  return request({
    url: `${USER_API}/agent/skill/acquired/download`,
    method: 'get',
    params,
    responseType: 'blob',
  });
};

// 删除我添加的skill
export const deleteAcquiredSkill = data => {
  return request({
    url: `${USER_API}/agent/skill/acquired`,
    method: 'delete',
    data,
  });
};

// 我添加的skill详情
export const getAcquiredSkillDetail = params => {
  return request({
    url: `${USER_API}/agent/skill/acquired/detail`,
    method: 'get',
    params,
  });
};

// 我添加的skill历史版本列表
export const getAcquiredSkillVersionList = params => {
  return request({
    url: `${USER_API}/agent/skill/acquired/version/list`,
    method: 'get',
    params,
  });
};

// 新增添加的skill变量配置
export const createAcquiredSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/acquired/config`,
    method: 'post',
    data,
  });
};

// 修改添加的skill变量配置
export const updateAcquiredSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/acquired/config`,
    method: 'put',
    data,
  });
};

// 删除添加的skill变量配置
export const deleteAcquiredSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/acquired/config`,
    method: 'delete',
    data,
  });
};
