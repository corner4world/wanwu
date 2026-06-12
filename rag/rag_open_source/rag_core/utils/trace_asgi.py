import logging
import time

logger = logging.getLogger(__name__)


class TraceLoggingMiddleware:
    """FastAPI/ASGI access log 中间件。

    使用纯 ASGI 实现，避免 Starlette BaseHTTPMiddleware 缓冲 SSE 流式响应导致
    问答接口失效的问题。trace_id / span_id 由 logging_config.TraceIdFilter 在
    所有日志行上自动注入，本中间件只负责输出 access log（cost / status / 路径）。
    """

    def __init__(self, app):
        self.app = app

    async def __call__(self, scope, receive, send):
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        start_time = time.time()
        method = scope.get("method", "-")
        path = scope.get("path", "-")
        query_string = scope.get("query_string", b"").decode("utf-8", errors="ignore")
        full_path = f"{path}?{query_string}" if query_string else path

        if "/apidocs" in full_path or "/docs" in full_path or "/openapi.json" in full_path:
            await self.app(scope, receive, send)
            return

        status_holder = {"code": 0}

        async def send_wrapper(message):
            if message.get("type") == "http.response.start":
                status_holder["code"] = message.get("status", 0)
            await send(message)

        try:
            await self.app(scope, receive, send_wrapper)
        finally:
            # access log 仅用于观察，任何异常都不得影响下游 ASGI 行为
            try:
                cost = round((time.time() - start_time) * 1000, 2)
                status = status_holder["code"]

                # trace_id / span_id 已由 logging_config.TraceIdFilter 统一注入，此处不再重复拼接
                log_msg = (
                    f"{cost}ms | {status} | "
                    f"{method} | {full_path}"
                )
                if status and status < 400:
                    logger.info(log_msg)
                else:
                    logger.error(log_msg)
            except Exception:
                logger.exception("TraceLoggingMiddleware access log failed (response untouched)")
