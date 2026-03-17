from fastapi import APIRouter

from controllers.auth import router as auth_router
from controllers.me import router as me_router
from controllers.healthCheck import router as health_router

router = APIRouter()

router.include_router(auth_router, prefix="/auth", tags=["auth"])
router.include_router(me_router, prefix="/me", tags=["me"])
router.include_router(health_router, tags=["health"])
