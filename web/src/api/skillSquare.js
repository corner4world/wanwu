// @description: skill广场相关接口
import request from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';

// 获取skill广场列表
export const getSquareSkillList = data => {
  return request({
    url: `${USER_API}/square/skill/list`,
    method: 'get',
    params: data,
  });
};

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
// 获取skill广场详情
export const getSquareSkillDetail = data => {
  return request({
    url: `${USER_API}/square/skill/detail`,
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
