#from flask import jsonify
#
#def health_check():
#    return jsonify({
#                'Status':'Ok',
#                'Code':'200'
#        }), 200

from fastapi import APIRouter

router = APIRouter()


@router.get("/health")
async def health() -> dict:
    return {'Status':'Ok','Code':'200'}
