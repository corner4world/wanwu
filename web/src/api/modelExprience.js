import service from '@/utils/request';
import { MODEL_API } from '@/utils/requestConstants';

// 获取模型体验历史对话记录
export const fetchChatList = (params = {}) => {
  return service({
    url: `${MODEL_API}/model/experience/dialogs`,
    method: 'get',
    params,
  });
};
// 新建/保存对话
export const createAndUpdateChat = (data = {}) => {
  return service({
    url: `${MODEL_API}/model/experience/dialog`,
    method: 'post',
    data,
  });
};
// 删除对话
export const deleteChat = (data = {}) => {
  return service({
    url: `${MODEL_API}/model/experience/dialog`,
    method: 'delete',
    data,
  });
};
// 提取文件
export const extractFile = (data = {}) => {
  return service({
    url: `${MODEL_API}/model/experience/file/extract`,
    method: 'post',
    data,
  });
};

// 获取问答详情
export const getExprienceDetail = (params = {}) => {
  return service({
    url: `${MODEL_API}/model/experience/dialog/records`,
    method: 'get',
    params,
  });
};

// 模型体验流式问答主动停止
export const cancelModelExperienceStream = (data = {}) => {
  return service({
    url: `${MODEL_API}/model/experience/llm/cancel`,
    method: 'post',
    data,
  });
};
