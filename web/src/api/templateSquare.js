import request from '@/utils/request';
import { USER_API } from '@/utils/requestConstants';

/*---工作流模板---*/
export const getWorkflowTempList = data => {
  return request({
    url: `${USER_API}/workflow/template/list`,
    method: 'get',
    params: data,
  });
};
export const getWorkflowTempInfo = data => {
  return request({
    url: `${USER_API}/workflow/template/detail`,
    method: 'get',
    params: data,
  });
};
export const getWorkflowRecommendsList = data => {
  return request({
    url: `${USER_API}/workflow/template/recommend`,
    method: 'get',
    params: data,
  });
};
export const downloadWorkflow = params => {
  return request({
    url: `${USER_API}/workflow/template/download`,
    method: 'get',
    params,
    responseType: 'blob',
  });
};
export const copyWorkflowTemplate = data => {
  return request({
    url: `${USER_API}/workflow/template`,
    method: 'post',
    data,
  });
};

/*---提示词模板---*/
export const getPromptTempList = data => {
  return request({
    url: `${USER_API}/prompt/template/list`,
    method: 'get',
    params: data,
  });
};

export const copyPromptTemplate = data => {
  return request({
    url: `${USER_API}/prompt/template`,
    method: 'post',
    data,
  });
};

/*---自定义提示词---*/
export const getCustomPromptList = data => {
  return request({
    url: `${USER_API}/prompt/custom/list`,
    method: 'get',
    params: data,
  });
};

export const createCustomPrompt = data => {
  return request({
    url: `${USER_API}/prompt/custom`,
    method: 'post',
    data,
  });
};

export const editCustomPrompt = data => {
  return request({
    url: `${USER_API}/prompt/custom`,
    method: 'put',
    data,
  });
};

export const copyCustomPrompt = data => {
  return request({
    url: `${USER_API}/prompt/custom/copy`,
    method: 'post',
    data,
  });
};

export const deleteCustomPrompt = data => {
  return request({
    url: `${USER_API}/prompt/custom`,
    method: 'delete',
    data,
  });
};

/*---Skills---*/

// 获取自定义skills列表
export const getCustomSkillList = data => {
  return request({
    url: `${USER_API}/agent/skill/custom/list`,
    method: 'get',
    params: data,
  });
};

// 删除自定义skills
export const deleteCustomSkill = data => {
  return request({
    url: `${USER_API}/agent/skill/custom`,
    method: 'delete',
    data,
  });
};

// 查询自定义skills详情
export const getCustomSkillInfo = data => {
  return request({
    url: `${USER_API}/agent/skill/custom/detail`,
    method: 'get',
    params: data,
  });
};

// 新增自定义skill配置
export const createCustomSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/custom/config`,
    method: 'post',
    data,
  });
};

// 修改自定义skill配置
export const updateCustomSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/custom/config`,
    method: 'put',
    data,
  });
};

// 删除自定义skill配置
export const deleteCustomSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/custom/config`,
    method: 'delete',
    data,
  });
};

// 创建自定义skills会话
export const createCustomSkillConversation = data => {
  return request({
    url: `${USER_API}/agent/skill/conversation`,
    method: 'post',
    data,
  });
};

// 删除自定义skill会话
export const delCustomSkillConversation = data => {
  return request({
    url: `${USER_API}/agent/skill/conversation`,
    method: 'delete',
    data,
  });
};

// 查询自定义skill会话列表
export const getCustomSkillConversationList = data => {
  return request({
    url: `${USER_API}/agent/skill/conversation/list`,
    method: 'get',
    params: data,
  });
};

// 查询自定义skill会话详情
export const getCustomSkillConversationDetail = data => {
  return request({
    url: `${USER_API}/agent/skill/conversation/detail`,
    method: 'get',
    params: data,
  });
};

// 自定义skill会话sse
export const getCustomSkillSSeUrl = () => {
  return `${USER_API}/agent/skill/conversation/chat`;
};

// 发送自定义skill到资源库
export const sendCustomSkillToResource = data => {
  return request({
    url: `${USER_API}/agent/skill/conversation/save`,
    method: 'post',
    data,
  });
};

// 创建自定义skills
export const createCustomSkill = data => {
  return request({
    url: `${USER_API}/agent/skill/custom`,
    method: 'post',
    data,
  });
};

// 校验自定义skills
export const checkCustomSkill = data => {
  return request({
    url: `${USER_API}/agent/skill/custom/check`,
    method: 'post',
    data,
  });
};

// 清空skillChat对话
export const clearSkillConversation = data => {
  return request({
    url: `${USER_API}/agent/skill/conversation/clear`,
    method: 'delete',
    data,
  });
};

// 获取skill选择列表（包含内置|自定义）
export const getSkillSelectList = data => {
  return request({
    url: `${USER_API}/agent/skill/select`,
    method: 'get',
    params: data,
  });
};

// 获取内置skills列表
export const getResourceBuiltinSkillList = data => {
  return request({
    url: `${USER_API}/agent/skill/builtin/list`,
    method: 'get',
    params: data,
  });
};

// 获取内置skills详情
export const getResourceBuiltinSkillDetail = data => {
  return request({
    url: `${USER_API}/agent/skill/builtin/detail`,
    method: 'get',
    params: data,
  });
};

// 内置skil下载
export const downloadBuiltinSkill = data => {
  return request({
    url: `${USER_API}/builtin/skill/download`,
    method: 'get',
    params: data,
    responseType: 'blob',
  });
};

// 新增内置skill配置
export const createResourceBuiltinSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/builtin/config`,
    method: 'post',
    data,
  });
};

// 编辑内置skill配置
export const updateResourceBuiltinSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/builtin/config`,
    method: 'put',
    data,
  });
};

// 删除内置skill配置
export const deleteResourceBuiltinSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/skill/builtin/config`,
    method: 'delete',
    data,
  });
};

// 新增添加的skill配置
export const createAcquiredSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/acquired/skill/config`,
    method: 'post',
    data,
  });
};

// 修改添加的skill配置
export const updateAcquiredSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/acquired/skill/config`,
    method: 'put',
    data,
  });
};

// 删除添加的skill配置
export const deleteAcquiredSkillConfig = data => {
  return request({
    url: `${USER_API}/agent/acquired/skill/config`,
    method: 'delete',
    data,
  });
};
