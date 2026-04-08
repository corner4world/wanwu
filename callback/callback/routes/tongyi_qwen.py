import http

from flask import g, jsonify, request

from callback.services import ali_multi_modal as ali_service
from callback.utils.decorators import require_bearer_auth
from utils.response import BizError

from . import callback_bp

generator = ali_service.AliGenAI()


@callback_bp.route("/qwen-t2i/qwen-image-max", methods=["POST"])
@require_bearer_auth
def qwen_image_max():
    """
    通义千问文生图:qwen-image-max
    ---
    tags:
      - Tongyi Qwen
    parameters:
      - in: header
        name: Authorization
        schema:
          type: string
        required: true
        description: 认证 Token (格式 Bearer <token>)
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - prompt
            properties:
              prompt:
                type: string
                description: 正向提示词,用于描述期望生成的图像内容、风格和构图。支持中英文,长度不超过800个字符.
                example: "一只在太空中飞翔的猫，赛博朋克风格"
              negative_prompt:
                type: string
                description: 反向提示词,用于描述不希望在图像中出现的内容,对画面进行限制。支持中英文,长度不超过500个字符
                default: ""
                example: "模糊，低质量"
              size:
                type: string
                description: 输出图像的分辨率，格式为宽*高。默认分辨率为1664*928,可选分辨率及对应比例为1664*928(默认,16:9)、1472*1104(4:3)、1328*1328(1:1)、1104*1472(3:4)和 928*1664(9:16)。
                default: "1664*928"
                example: "1664*928"
              n:
                type: integer
                description: 生成数量
                default: 1
                example: 1
    responses:
      200:
        description: 生成任务提交成功/生成成功
        content:
          application/json:
            schema:
              type: object
              description: 返回生成结果
    """
    data = request.get_json()
    prompt = data.get("prompt")
    if not prompt:
        raise BizError("missing prompt", code=http.HTTPStatus.BAD_REQUEST)
    negative_prompt = data.get("negative_prompt", "")
    size = data.get("size", "1664*928")
    n = data.get("n", 1)

    res = generator.qwen_text_to_image(
        api_key=g.api_key,
        prompt=prompt,
        model="qwen-image-max",
        negative_prompt=negative_prompt,
        size=size,
        n=n,
    )
    return jsonify(res)


@callback_bp.route("/qwen-i2i/qwen-image-edit-max", methods=["POST"])
@require_bearer_auth
def qwen_image_edit_max():
    """
    通义千问图片编辑: qwen-image-edit-max
    ---
    tags:
      - Tongyi Qwen
    summary: 调用通义千问进行图片编辑 (Image-to-Image)
    parameters:
      - in: header
        name: Authorization
        schema:
          type: string
        required: true
        description: 认证 Token (格式 Bearer <token>)
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - prompt
              - images
            properties:
              prompt:
                type: string
                description: 编辑指令 (正向提示词), 用于描述期望对原图进行的修改内容。
                example: "生成一张符合深度图的图像，遵循以下描述：一辆红色的破旧的自行车停在一条泥泞的小路上，背景是茂密的原始森林"
              images:
                type: array
                items:
                  type: string
                description: 输入图像的 URL 或 Base64 编码数据。支持传入1-3张图像。多图输入时,按照数组顺序定义图像顺序
                example: ["https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20250925/fpakfo/image36.webp"]
              negative_prompt:
                type: string
                description: 反向提示词, 用于描述不希望在图像中出现的内容。
                default: ""
                example: "模糊，低质量，变形"
              size:
                type: string
                description: 输出图像分辨率格式为"宽*高"（如"1024*1536"，宽高范围[512,2048]），常见比例推荐为：1:1（1024*1024、1536*1536）、2:3（768*1152、1024*1536）、3:2（1152*768、1536*1024）、3:4（960*1280、1080*1440）、4:3（1280*960、1440*1080）、9:16（720*1280、1080*1920）、16:9（1280*720、1920*1080）以及 21:9（1344*576、2048*872）。
                default: "1024*1024"
                example: "1024*1024"
              n:
                type: integer
                description: 生成数量
                default: 1
                example: 1
    responses:
      200:
        description: 生成任务提交成功/生成成功
        content:
          application/json:
            schema:
              type: object
              description: 返回生成结果 (通常包含 task_id 或生成的图片地址)
    """
    data = request.get_json()
    prompt = data.get("prompt")
    if not prompt:
        raise BizError("missing prompt", code=http.HTTPStatus.BAD_REQUEST)
    negative_prompt = data.get("negative_prompt", "")
    size = data.get("size")
    n = data.get("n", 1)
    images = data.get("images")
    if not images or not isinstance(images, list):
        raise BizError("missing images", code=http.HTTPStatus.BAD_REQUEST)

    res = generator.image_to_image_generate(
        api_key=g.api_key,
        prompt=prompt,
        model="qwen-image-edit-max",
        images=images,
        negative_prompt=negative_prompt,
        size=size,
        n=n,
    )
    return jsonify(res)


@callback_bp.route("/qwen-image/qwen-image-2.0", methods=["POST"])
@require_bearer_auth
def qwen_image_2_0():
    """
    通义千问图片编辑: qwen-image-2.0
    ---
    tags:
      - Tongyi Qwen
    summary: 通义千问进行文生图和图片编辑
    parameters:
      - in: header
        name: Authorization
        schema:
          type: string
        required: true
        description: 认证 Token (格式 Bearer <token>)
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - prompt
            properties:
              prompt:
                type: string
                description: 编辑指令 (正向提示词), 用于描述期望对原图进行的修改内容。
                example: "生成一张符合深度图的图像，遵循以下描述：一辆红色的破旧的自行车停在一条泥泞的小路上，背景是茂密的原始森林"
              images:
                type: array
                items:
                  type: string
                description: 输入图像的 URL 或 Base64 编码数据。支持传入1-3张图像。多图输入时,按照数组顺序定义图像顺序
                example: ["https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20250925/fpakfo/image36.webp"]
              negative_prompt:
                type: string
                description: 反向提示词, 用于描述不希望在图像中出现的内容。
                default: ""
                example: "模糊，低质量，变形"
              size:
                type: string
                description: 输出图像分辨率格式为"宽*高"（如"1024*1536"，宽高范围[512,2048]），常见比例推荐为：1:1（1024*1024、1536*1536）、2:3（768*1152、1024*1536）、3:2（1152*768、1536*1024）、3:4（960*1280、1080*1440）、4:3（1280*960、1440*1080）、9:16（720*1280、1080*1920）、16:9（1280*720、1920*1080）以及 21:9（1344*576、2048*872）。
                default: "1024*1024"
                example: "1024*1024"
              n:
                type: integer
                description: 生成数量
                default: 1
                example: 1
    responses:
      200:
        description: 生成任务提交成功/生成成功
        content:
          application/json:
            schema:
              type: object
              description: 返回生成结果 (通常包含 task_id 或生成的图片地址)
    """
    data = request.get_json()
    prompt = data.get("prompt")
    if not prompt:
        raise BizError("missing prompt", code=http.HTTPStatus.BAD_REQUEST)
    negative_prompt = data.get("negative_prompt", "")
    size = data.get("size")
    n = data.get("n", 1)
    images = data.get("images")

    res = generator.image_to_image_generate(
        api_key=g.api_key,
        prompt=prompt,
        model="qwen-image-2.0",
        images=images,
        negative_prompt=negative_prompt,
        size=size,
        n=n,
    )
    return jsonify(res)


@callback_bp.route("/qwen-image/qwen-image-2.0-pro", methods=["POST"])
@require_bearer_auth
def qwen_image_2_0_pro():
    """
    通义千问图片编辑: qwen-image-2.0-pro
    ---
    tags:
      - Tongyi Qwen
    summary: 通义千问进行文生图和图片编辑
    parameters:
      - in: header
        name: Authorization
        schema:
          type: string
        required: true
        description: 认证 Token (格式 Bearer <token>)
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required:
              - prompt
            properties:
              prompt:
                type: string
                description: 编辑指令 (正向提示词), 用于描述期望对原图进行的修改内容。
                example: "生成一张符合深度图的图像，遵循以下描述：一辆红色的破旧的自行车停在一条泥泞的小路上，背景是茂密的原始森林"
              images:
                type: array
                items:
                  type: string
                description: 输入图像的 URL 或 Base64 编码数据。支持传入1-3张图像。多图输入时,按照数组顺序定义图像顺序
                example: ["https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20250925/fpakfo/image36.webp"]
              negative_prompt:
                type: string
                description: 反向提示词, 用于描述不希望在图像中出现的内容。
                default: ""
                example: "模糊，低质量，变形"
              size:
                type: string
                description: 输出图像分辨率格式为"宽*高"（如"1024*1536"，宽高范围[512,2048]），常见比例推荐为：1:1（1024*1024、1536*1536）、2:3（768*1152、1024*1536）、3:2（1152*768、1536*1024）、3:4（960*1280、1080*1440）、4:3（1280*960、1440*1080）、9:16（720*1280、1080*1920）、16:9（1280*720、1920*1080）以及 21:9（1344*576、2048*872）。
                default: "1024*1024"
                example: "1024*1024"
              n:
                type: integer
                description: 生成数量
                default: 1
                example: 1
    responses:
      200:
        description: 生成任务提交成功/生成成功
        content:
          application/json:
            schema:
              type: object
              description: 返回生成结果 (通常包含 task_id 或生成的图片地址)
    """
    data = request.get_json()
    prompt = data.get("prompt")
    if not prompt:
        raise BizError("missing prompt", code=http.HTTPStatus.BAD_REQUEST)
    negative_prompt = data.get("negative_prompt", "")
    size = data.get("size")
    n = data.get("n", 1)
    images = data.get("images")

    res = generator.image_to_image_generate(
        api_key=g.api_key,
        prompt=prompt,
        model="qwen-image-2.0-pro",
        images=images,
        negative_prompt=negative_prompt,
        size=size,
        n=n,
    )
    return jsonify(res)
