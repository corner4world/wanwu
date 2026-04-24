// @description: 我添加的skill相关接口
import request from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';

// 获取skill列表
export const getJoinerSkillList = data => {
  return request({
    url: `${USER_API}/agent/joiner/skills`,
    method: 'get',
    params: data,
  });
};

// 删除skill
export const deleteJoinerSkill = data => {
  return request({
    url: `${USER_API}/agent/joiner/skills`,
    method: 'delete',
    data,
  });
};

// skill详情
export const getJoinerSkillDetail = data => {
  return request({
    url: `${USER_API}/agent/joiner/skills/detail`,
    method: 'get',
    params: data,
  });
};
