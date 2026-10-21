import { jsonBody, request } from './client';
import {
  makeMockSlots, mockAdmin, mockAnalyses, mockConsultants, mockStudent,
  mockConsultations,
} from './mock';
import type {
  AuthResponse, Consultation, ConsultationSlot, Consultant, ListResponse,
  NotificationConsentResponse, ResumeAnalysis, UploadedFile, User,
} from './types';

const MOCK = import.meta.env.VITE_MOCK_MODE === 'true';
let demoConsultations = [...mockConsultations];
let demoSlots = mockConsultants.flatMap((consultant) => makeMockSlots(consultant.id));
let demoAnalyses: ResumeAnalysis[] = [...mockAnalyses];
let demoConsent = false;
const delay = async <T,>(value: T): Promise<T> => value;

export const authApi = {
  async loginWithVK(launchParams: string): Promise<AuthResponse> {
    return request<AuthResponse>('/auth/vk', { method: 'POST', body: jsonBody({ launch_params: launchParams }) });
  },
  me: () => request<User>('/me'),
  async consent(enabled: boolean): Promise<NotificationConsentResponse> {
    if (MOCK) { demoConsent = enabled; return delay({ notification_consent: demoConsent }); }
    return request<NotificationConsentResponse>('/me/notification-consent', { method: 'POST', body: jsonBody({ enabled }) });
  },
};

export const consultantsApi = {
  async list(): Promise<Consultant[]> {
    if (MOCK) return delay(mockConsultants.filter((item) => item.is_active));
    const response = await request<ListResponse<Consultant>>('/consultants');
    return response.items;
  },
  get: (id: string) => request<Consultant>(`/consultants/${id}`),
  async slots(id: string, from: Date, to: Date): Promise<ConsultationSlot[]> {
    if (MOCK) return delay(demoSlots.filter((slot) => slot.consultant_id === id && slot.status === 'available' && new Date(slot.starts_at) >= from && new Date(slot.starts_at) <= to));
    const params = new URLSearchParams({ from: from.toISOString(), to: to.toISOString() });
    const response = await request<ListResponse<ConsultationSlot>>(`/consultants/${id}/slots?${params.toString()}`);
    return response.items;
  },
};

export const consultationsApi = {
  async create(input: { slot_id: string; problem_description: string; resume_file_id?: string | null }): Promise<Consultation> {
    if (MOCK) {
      const slot = demoSlots.find((item) => item.id === input.slot_id && item.status === 'available');
      if (!slot) throw new Error('Выбранное время уже занято. Обновите расписание.');
      const consultant = mockConsultants.find((item) => item.id === slot.consultant_id);
      const created: Consultation = {
        id: crypto.randomUUID(), student_id: mockStudent.id, consultant_id: slot.consultant_id, slot_id: slot.id,
        starts_at: slot.starts_at, ends_at: slot.ends_at, problem_description: input.problem_description,
        resume_file_id: input.resume_file_id ?? null, status: 'pending', reschedule_count: 0,
        consultant_name: consultant ? `${consultant.first_name} ${consultant.last_name}` : 'Консультант',
        created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
      };
      demoConsultations = [created, ...demoConsultations];
      demoSlots = demoSlots.map((item) => item.id === slot.id ? { ...item, status: 'booked' } : item);
      return delay(created);
    }
    return request<Consultation>('/consultations', { method: 'POST', body: jsonBody(input) });
  },
  async mine(): Promise<Consultation[]> {
    if (MOCK) return delay(demoConsultations);
    const response = await request<ListResponse<Consultation>>('/consultations/my');
    return response.items;
  },
  get: (id: string) => request<Consultation>(`/consultations/${id}`),
  async reschedule(id: string, newSlotId: string): Promise<Consultation> {
    if (MOCK) {
      const consultation = demoConsultations.find((item) => item.id === id);
      const slot = demoSlots.find((item) => item.id === newSlotId && item.status === 'available');
      if (!consultation || !slot) throw new Error('Запись или выбранный слот недоступны.');
      if (consultation.reschedule_count >= 1) throw new Error('RESCHEDULE_LIMIT_REACHED');
      demoSlots = demoSlots.map((item) => item.id === consultation.slot_id ? { ...item, status: 'available' } : item.id === slot.id ? { ...item, status: 'booked' } : item);
      const updated = { ...consultation, slot_id: slot.id, starts_at: slot.starts_at, ends_at: slot.ends_at, reschedule_count: consultation.reschedule_count + 1, updated_at: new Date().toISOString() };
      demoConsultations = demoConsultations.map((item) => item.id === id ? updated : item);
      return delay(updated);
    }
    return request<Consultation>(`/consultations/${id}/reschedule`, { method: 'POST', body: jsonBody({ new_slot_id: newSlotId }) });
  },
  async cancel(id: string): Promise<void> {
    if (MOCK) {
      demoConsultations = demoConsultations.map((item) => item.id === id ? { ...item, status: 'cancelled', updated_at: new Date().toISOString() } : item);
      demoSlots = demoSlots.map((slot) => slot.id === demoConsultations.find((item) => item.id === id)?.slot_id ? { ...slot, status: 'available' } : slot);
      return;
    }
    return request<void>(`/consultations/${id}/cancel`, { method: 'POST' });
  },
};

export const adminApi = {
  async consultations(params: { status?: string; from?: Date; to?: Date } = {}): Promise<Consultation[]> {
    if (MOCK) return delay(demoConsultations.filter((item) => (!params.status || item.status === params.status) && (!params.from || new Date(item.starts_at) >= params.from) && (!params.to || new Date(item.starts_at) <= params.to)));
    const query = new URLSearchParams({ limit: '100', offset: '0' });
    if (params.status) query.set('status', params.status);
    if (params.from) query.set('from', params.from.toISOString());
    if (params.to) query.set('to', params.to.toISOString());
    const response = await request<ListResponse<Consultation>>(`/admin/consultations?${query.toString()}`);
    return response.items;
  },
  async confirm(id: string): Promise<void> {
    if (MOCK) { demoConsultations = demoConsultations.map((item) => item.id === id ? { ...item, status: 'confirmed', updated_at: new Date().toISOString() } : item); return; }
    return request<void>(`/admin/consultations/${id}/confirm`, { method: 'POST' });
  },
  async complete(id: string): Promise<void> {
    if (MOCK) { demoConsultations = demoConsultations.map((item) => item.id === id ? { ...item, status: 'completed', updated_at: new Date().toISOString() } : item); return; }
    return request<void>(`/admin/consultations/${id}/complete`, { method: 'POST' });
  },
  async slots(from: Date, to: Date): Promise<ConsultationSlot[]> {
    if (MOCK) return delay(demoSlots.filter((slot) => new Date(slot.starts_at) >= from && new Date(slot.starts_at) <= to));
    const params = new URLSearchParams({ from: from.toISOString(), to: to.toISOString() });
    const response = await request<ListResponse<ConsultationSlot>>(`/admin/slots?${params.toString()}`);
    return response.items;
  },
  async createSlot(input: { starts_at: string; ends_at: string }): Promise<ConsultationSlot> {
    if (MOCK) {
      const created: ConsultationSlot = { id: crypto.randomUUID(), consultant_id: mockConsultants[0].id, ...input, status: 'available' };
      demoSlots = [...demoSlots, created]; return delay(created);
    }
    return request<ConsultationSlot>('/admin/slots', { method: 'POST', body: jsonBody(input) });
  },
  async updateSlot(id: string, input: { starts_at: string; ends_at: string }): Promise<ConsultationSlot> {
    if (MOCK) {
      demoSlots = demoSlots.map((slot) => slot.id === id ? { ...slot, ...input } : slot);
      const item = demoSlots.find((slot) => slot.id === id);
      if (!item) throw new Error('Слот не найден');
      return delay(item);
    }
    return request<ConsultationSlot>(`/admin/slots/${id}`, { method: 'PATCH', body: jsonBody(input) });
  },
  async blockSlot(id: string): Promise<void> {
    if (MOCK) { demoSlots = demoSlots.map((slot) => slot.id === id ? { ...slot, status: 'blocked' } : slot); return; }
    return request<void>(`/admin/slots/${id}/block`, { method: 'POST' });
  },
  async profile(): Promise<Consultant> {
    if (MOCK) return delay({ ...mockConsultants[0], user_id: mockAdmin.id });
    return request<Consultant>('/admin/profile');
  },
  async updateProfile(input: { specialization: string; description: string }): Promise<Consultant> {
    if (MOCK) return delay({ ...mockConsultants[0], ...input, user_id: mockAdmin.id });
    return request<Consultant>('/admin/profile', { method: 'PATCH', body: jsonBody(input) });
  },
};

export const filesApi = {
  async uploadResume(file: File): Promise<UploadedFile> {
    if (MOCK) return delay({ id: crypto.randomUUID(), original_filename: file.name, mime_type: file.type || 'application/octet-stream', size_bytes: file.size, created_at: new Date().toISOString() });
    const form = new FormData(); form.append('file', file);
    return request<UploadedFile>('/files/resumes', { method: 'POST', body: form });
  },
  download: async (id: string): Promise<Blob> => request<Blob>(`/files/${id}`),
  async remove(id: string): Promise<void> {
    if (MOCK) return;
    return request<void>(`/files/${id}`, { method: 'DELETE' });
  },
};

export const resumeAnalysisApi = {
  async create(fileId: string, externalAIConsent: boolean): Promise<ResumeAnalysis> {
    if (MOCK) {
      const task: ResumeAnalysis = { id: crypto.randomUUID(), file_id: fileId, status: 'processing', attempt_count: 1, created_at: new Date().toISOString(), updated_at: new Date().toISOString() };
      demoAnalyses = [task, ...demoAnalyses];
      window.setTimeout(() => {
        demoAnalyses = demoAnalyses.map((item) => item.id === task.id ? { ...item, status: 'completed', result: { summary: 'Демонстрационный результат. В реальном режиме этот отчёт формирует настроенный AI-провайдер на сервере.', score: 78, strengths: ['Есть конкретные проекты', 'Указан основной технологический стек'], weaknesses: ['Часть достижений описана без измеримого результата'], recommendations: ['Опишите вклад в каждый проект', 'Добавьте конкретные результаты и цифры', 'Сгруппируйте навыки по категориям'] }, completed_at: new Date().toISOString(), updated_at: new Date().toISOString() } : item);
      }, 1800);
      return delay(task);
    }
    return request<ResumeAnalysis>('/resume-analyses', { method: 'POST', body: jsonBody({ file_id: fileId, external_ai_consent: externalAIConsent }) });
  },
  async list(): Promise<ResumeAnalysis[]> {
    if (MOCK) return delay(demoAnalyses);
    const response = await request<ListResponse<ResumeAnalysis>>('/resume-analyses');
    return response.items;
  },
  async get(id: string): Promise<ResumeAnalysis> {
    if (MOCK) {
      const analysis = demoAnalyses.find((item) => item.id === id);
      if (!analysis) throw new Error('Анализ не найден');
      return delay(analysis);
    }
    return request<ResumeAnalysis>(`/resume-analyses/${id}`);
  },
};

export const isMockMode = MOCK;
