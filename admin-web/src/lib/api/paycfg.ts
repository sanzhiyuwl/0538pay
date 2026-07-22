/** 支付方式 / 微信公众号小程序 / 企业微信 API（对齐后端 /admin/paytypes|weixins|weworks）。 */
import { request } from './client'

// ===== 支付方式 pay_type =====
export interface PayTypeView {
  id: number
  name: string
  showname: string
  device: number // 0=PC+Mobile 1=PC 2=Mobile
  today: string
  status: number
}
export interface PayTypeSaveReq {
  name: string
  showname: string
  device: number
}
export function fetchPayTypes(): Promise<{ list: PayTypeView[] }> {
  return request<{ list: PayTypeView[] }>('/admin/paytypes')
}
export function createPayType(body: PayTypeSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>('/admin/paytypes', { method: 'POST', body })
}
export function updatePayType(id: number, body: PayTypeSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/paytypes/${id}`, { method: 'PUT', body })
}
export function setPayTypeStatus(id: number, status: number): Promise<{ id: number; status: number }> {
  return request<{ id: number; status: number }>(`/admin/paytypes/${id}/status`, { method: 'PUT', body: { status } })
}
export function deletePayType(id: number): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/paytypes/${id}`, { method: 'DELETE' })
}

// ===== 微信公众号/小程序 pay_weixin =====
export interface WeixinView {
  id: number
  type: number // 0=服务号 1=小程序
  name: string
  appid: string
  appsecret: string // 脱敏
}
export interface WeixinSaveReq {
  type: number
  name: string
  appid: string
  appsecret: string
}
export function fetchWeixins(): Promise<{ list: WeixinView[] }> {
  return request<{ list: WeixinView[] }>('/admin/weixins')
}
export function createWeixin(body: WeixinSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>('/admin/weixins', { method: 'POST', body })
}
export function updateWeixin(id: number, body: WeixinSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/weixins/${id}`, { method: 'PUT', body })
}
export function deleteWeixin(id: number): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/weixins/${id}`, { method: 'DELETE' })
}

// ===== 企业微信 pay_wework =====
export interface WeworkView {
  id: number
  name: string
  appid: string
  appsecret: string // 脱敏
  kfnum: number
  status: number
}
export interface WeworkSaveReq {
  name: string
  appid: string
  appsecret: string
}
export function fetchWeworks(): Promise<{ list: WeworkView[] }> {
  return request<{ list: WeworkView[] }>('/admin/weworks')
}
export function createWework(body: WeworkSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>('/admin/weworks', { method: 'POST', body })
}
export function updateWework(id: number, body: WeworkSaveReq): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/weworks/${id}`, { method: 'PUT', body })
}
export function setWeworkStatus(id: number, status: number): Promise<{ id: number; status: number }> {
  return request<{ id: number; status: number }>(`/admin/weworks/${id}/status`, { method: 'PUT', body: { status } })
}
export function deleteWework(id: number): Promise<{ id: number }> {
  return request<{ id: number }>(`/admin/weworks/${id}`, { method: 'DELETE' })
}
