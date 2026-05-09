// @description: 我添加的skill相关接口
import request from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';

// 获取skill列表
export const getAcquiredSkillList = data => {
  return request({
    url: `${USER_API}/agent/acquired/skill/list`,
    method: 'get',
    params: data,
  });
};

// 删除skill
export const deleteAcquiredSkill = data => {
  return request({
    url: `${USER_API}/agent/acquired/skill`,
    method: 'delete',
    data,
  });
};

// skill详情
export const getAcquiredSkillDetail = data => {
  return request({
    url: `${USER_API}/agent/acquired/skill/detail`,
    method: 'get',
    params: data,
  });
};
