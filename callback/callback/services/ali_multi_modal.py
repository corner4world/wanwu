# -*- coding: utf-8 -*-
import time
from typing import Dict, List, Optional

import dashscope
import requests
from dashscope import ImageSynthesis, MultiModalConversation, VideoSynthesis
from dashscope.aigc.image_generation import ImageGeneration
from dashscope.api_entities.dashscope_response import Message

from callback.utils.url_util import process_image_to_base64
from utils.log import logger


class AliGenAI:
    def __init__(self, region: str = "cn"):
        """
        初始化阿里云生图客户端
        :param region: 地域 ('cn' 为北京, 'intl' 为新加坡)
        """
        self._region = region
        self._setup_region(region)

    def _setup_region(self, region: str):
        if region == "intl":
            dashscope.base_http_api_url = "https://dashscope-intl.aliyuncs.com/api/v1"
        else:
            dashscope.base_http_api_url = "https://dashscope.aliyuncs.com/api/v1"

    def image_generate_legacy(
        self,
        api_key: str,
        prompt: str,
        images: Optional[List[str]] = None,
        model: str = "wan2.5-t2i-preview",
        negative_prompt: str = "",
        size: str = "1280*1280",
        n: int = 1,
    ):
        """
        调用 ImageSynthesis（旧版接口）
        适用模型：wan2.5及以下版本、qwen-image-plus、qwen-image
        """
        try:
            processed_images = None
            if images:
                processed_images = [process_image_to_base64(img) for img in images]
            rsp = ImageSynthesis.call(
                api_key=api_key,
                model=model,
                prompt=prompt,
                images=processed_images,
                negative_prompt=negative_prompt,
                n=n,
                size=size,
                prompt_extend=True,
                watermark=True,
            )
            return rsp
        except Exception as e:
            logger.exception(f"ImageSynthesis 调用失败: {e}")
            return None

    def image_to_image_generate(
        self,
        api_key: str,
        prompt: str,
        images: Optional[List[str]] = None,
        model: str = "qwen-image-plus",
        negative_prompt: str = "",
        size: str = "1280*1280",
        n: int = 1,
        prompt_extend: bool = True,
        watermark: bool = True,
    ):
        """
        调用 ImageGeneration（新版接口）
        支持文生图和图生图/编辑
        """
        content_list = []
        processed_images = None

        if images:
            processed_images = [process_image_to_base64(img) for img in images]
            for img in processed_images:
                content_list.append({"image": img})

        content_list.append({"text": prompt})
        # logger.info(f"image_generate content_list: {content_list}")

        message = Message(role="user", content=content_list)

        try:
            rsp = ImageGeneration.call(
                model=model,
                api_key=api_key,
                messages=[message],
                negative_prompt=negative_prompt,
                prompt_extend=prompt_extend,
                watermark=watermark,
                images=processed_images,
                n=n,
                size=size,
            )
            return rsp
        except Exception as e:
            logger.exception(f"ImageGeneration 调用失败: {e}")
            return None

    def text_to_image_generate(
        self,
        api_key: str,
        prompt: str,
        model: str = "wan2.6-t2i",
        negative_prompt: str = "",
        size: str = "1280*1280",
        n: int = 1,
        prompt_extend: bool = True,
        watermark: bool = True,
    ):
        """
        调用 ImageGeneration（新版接口）
        支持文生图和图生图/编辑
        """
        content_list = []

        content_list.append({"text": prompt})
        # logger.info(f"image_generate content_list: {content_list}")

        message = Message(role="user", content=content_list)

        try:
            rsp = ImageGeneration.call(
                model=model,
                api_key=api_key,
                messages=[message],
                negative_prompt=negative_prompt,
                prompt_extend=prompt_extend,
                watermark=watermark,
                n=n,
                size=size,
            )
            return rsp
        except Exception as e:
            logger.exception(f"ImageGeneration 调用失败: {e}")
            return None

    def qwen_text_to_image(
        self,
        api_key: str,
        prompt: str,
        model: str = "qwen-image-max",
        negative_prompt: str = "",
        size: str = "1280*1280",
        n: int = 1,
    ):
        """
        调用多模态对话接口
        """
        content_list = []

        content_list.append({"text": prompt})
        messages = [{"role": "user", "content": content_list}]

        try:
            rsp = MultiModalConversation.call(
                api_key=api_key,
                model=model,
                messages=messages,
                negative_prompt=negative_prompt,
                n=n,
                size=size,
                prompt_extend=True,
                watermark=True,
            )
            return rsp
        except Exception as e:
            logger.exception(f"MultiModalConversation 调用失败: {e}")
            return None

    def image_to_video_generate(
        self,
        api_key: str,
        img_url: str,
        prompt: Optional[str] = None,
        model: str = "wan2.6-i2v-flash",
        audio_url: Optional[str] = None,
        resolution: str = "720P",
        duration: int = 5,
        negative_prompt: str = "",
        shot_type: Optional[str] = None,
        template: Optional[str] = None,
    ):
        """
        图片生成视频
        """
        try:
            img_url = process_image_to_base64(img_url)
            resp = VideoSynthesis.call(
                api_key=api_key,
                model=model,
                prompt=prompt,
                img_url=img_url,
                audio_url=audio_url,
                resolution=resolution,
                duration=duration,
                prompt_extend=True,
                watermark=True,
                negative_prompt=negative_prompt,
                shot_type=shot_type,
                template=template,
            )
            return resp
        except Exception as e:
            logger.exception(f"ImageToVideo 调用失败: {e}")
            return None

    def first_and_last_image_to_video(
        self,
        api_key: str,
        first_frame_url: str,
        prompt: Optional[str] = None,
        model: str = "wan2.2-kf2v-flash",
        last_frame_url: Optional[str] = None,
        resolution: str = "720P",
        duration: int = 5,
        negative_prompt: str = "",
        template: Optional[str] = None,
    ):
        """
        首尾帧生成视频
        """
        try:
            first_frame_url = process_image_to_base64(first_frame_url)
            last_frame_url = process_image_to_base64(last_frame_url)
            resp = VideoSynthesis.call(
                api_key=api_key,
                model=model,
                prompt=prompt,
                first_frame_url=first_frame_url,
                last_frame_url=last_frame_url,
                resolution=resolution,
                duration=duration,
                prompt_extend=True,
                watermark=True,
                negative_prompt=negative_prompt,
                template=template,
            )
            return resp
        except Exception as e:
            logger.exception(f"FirstLastFrameVideo 调用失败: {e}")
            return None

    def text_to_video_generate(
        self,
        api_key: str,
        prompt: str,
        model: str = "wan2.6-t2v",
        audio_url: Optional[str] = None,
        size: str = "1280*720",
        duration: int = 5,
        negative_prompt: str = "",
        shot_type: Optional[str] = None,
    ):
        """
        文本生成视频
        """
        try:
            rsp = VideoSynthesis.call(
                api_key=api_key,
                model=model,
                prompt=prompt,
                audio_url=audio_url,
                size=size,
                duration=duration,
                negative_prompt=negative_prompt,
                prompt_extend=True,
                watermark=True,
                shot_type=shot_type,
            )
            # logger.info(f"text_to_video_generate response: {rsp}")
            return rsp
        except Exception as e:
            logger.exception(f"TextToVideo 调用失败: {e}")
            return None

    def image_to_video_sync(
        self,
        api_key: str,
        prompt: Optional[str] = None,
        first_frame_url: Optional[str] = None,
        last_frame_url: Optional[str] = None,
        first_clip_url: Optional[str] = None,
        audio_url: Optional[str] = None,
        model: str = "wan2.7-i2v",
        resolution: str = "720P",
        duration: int = 5,
        prompt_extend: bool = True,
        watermark: bool = True,
        poll_interval: int = 10,  # 轮询间隔，单位秒，默认10秒
        max_poll_time: int = 600,  # 最大轮询时间，单位秒，默认600秒
    ) -> Dict:
        """
        万相图生视频2.7同步接口
        支持首帧生视频、首尾帧生视频、视频续写

        :param api_key: API密钥
        :param prompt: 文本提示词
        :param first_frame_url: 首帧图像URL
        :param last_frame_url: 尾帧图像URL（可选，用于首尾帧生视频）
        :param first_clip_url: 首段视频URL（可选，用于视频续写）
        :param audio_url: 音频URL（可选）
        :param model: 模型名称，默认 wan2.7-i2v
        :param resolution: 分辨率，720P或1080P
        :param duration: 视频时长(秒)，范围2-15
        :param prompt_extend: 是否扩展提示词
        :param watermark: 是否添加水印
        :param poll_interval: 轮询间隔(秒)
        :param max_poll_time: 最大轮询时间(秒)
        :return: 包含视频URL的字典
        """
        base_url = dashscope.base_http_api_url
        headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "X-DashScope-Async": "enable",
        }

        media = []
        if first_frame_url:
            media.append(
                {"type": "first_frame", "url": process_image_to_base64(first_frame_url)}
            )
        if last_frame_url:
            media.append(
                {"type": "last_frame", "url": process_image_to_base64(last_frame_url)}
            )
        if first_clip_url:
            media.append({"type": "first_clip", "url": first_clip_url})
        if audio_url:
            media.append({"type": "driving_audio", "url": audio_url})

        payload = {
            "model": model,
            "input": {
                "prompt": prompt,
                "media": media,
            },
            "parameters": {
                "resolution": resolution,
                "duration": duration,
                "prompt_extend": prompt_extend,
                "watermark": watermark,
            },
        }

        try:
            create_url = f"{base_url}/services/aigc/video-generation/video-synthesis"
            response = requests.post(
                create_url, headers=headers, json=payload, timeout=30
            )
            response.raise_for_status()
            result = response.json()

            task_id = result.get("output", {}).get("task_id")
            if not task_id:
                logger.error(f"创建任务失败，未获取到task_id: {result}")
                return {"code": -1, "message": "创建任务失败", "data": result}

            query_url = f"{base_url}/tasks/{task_id}"
            query_headers = {
                "Authorization": f"Bearer {api_key}",
            }

            start_time = time.time()
            while time.time() - start_time < max_poll_time:
                time.sleep(poll_interval)

                query_response = requests.get(
                    query_url, headers=query_headers, timeout=30
                )
                query_response.raise_for_status()
                query_result = query_response.json()

                task_status = query_result.get("output", {}).get("task_status")
                logger.info(f"任务状态: {task_status}, task_id: {task_id}")

                if task_status == "SUCCEEDED":
                    video_url = query_result.get("output", {}).get("video_url")
                    return {
                        "code": 0,
                        "message": "success",
                        "data": {
                            "task_id": task_id,
                            "video_url": video_url,
                            "task_status": task_status,
                        },
                    }
                elif task_status == "FAILED":
                    error_msg = query_result.get("output", {}).get(
                        "message", "任务执行失败"
                    )
                    logger.error(f"任务执行失败: {error_msg}")
                    return {
                        "code": -1,
                        "message": error_msg,
                        "data": query_result,
                    }

            logger.error(f"任务超时，task_id: {task_id}")
            return {
                "code": -1,
                "message": f"任务超时，已等待{max_poll_time}秒",
                "data": {"task_id": task_id},
            }

        except Exception as e:
            logger.exception(f"ImageToVideoSync 调用失败: {e}")
            return {"code": -1, "message": str(e), "data": None}
