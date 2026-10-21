export type UserRole = 'student' | 'admin';
export type ConsultationStatus = 'pending' | 'confirmed' | 'completed' | 'cancelled';
export type SlotStatus = 'available' | 'booked' | 'blocked';
export type ResumeAnalysisStatus = 'queued' | 'processing' | 'completed' | 'failed';

export interface User {
  id: string;
  vk_id: number;
  first_name: string;
  last_name: string;
  avatar_url: string | null;
  role: UserRole;
  notification_consent?: boolean;
  notification_consent_at?: string | null;
}

export interface Consultant {
  id: string;
  user_id: string;
  first_name: string;
  last_name: string;
  avatar_url: string | null;
  specialization: string;
  description: string;
  is_active: boolean;
}

export interface ConsultationSlot {
  id: string;
  consultant_id: string;
  starts_at: string;
  ends_at: string;
  status: SlotStatus;
  created_at?: string;
  updated_at?: string;
}

export interface Consultation {
  id: string;
  student_id: string;
  consultant_id: string;
  slot_id: string;
  starts_at: string;
  ends_at: string;
  problem_description: string;
  resume_file_id?: string | null;
  status: ConsultationStatus;
  reschedule_count: number;
  consultant_name?: string;
  student_name?: string;
  created_at: string;
  updated_at: string;
}

export interface UploadedFile {
  id: string;
  original_filename: string;
  mime_type: string;
  size_bytes: number;
  created_at: string;
}

export interface ResumeAnalysisResult {
  summary?: string;
  score?: number | null;
  strengths?: string[];
  weaknesses?: string[];
  recommendations?: string[];
  structure_advice?: string[];
  project_advice?: string[];
  missing_information?: string[];
  [key: string]: unknown;
}

export interface ResumeAnalysis {
  id: string;
  file_id: string;
  status: ResumeAnalysisStatus;
  result?: ResumeAnalysisResult | string | null;
  error_code?: string | null;
  attempt_count: number;
  created_at: string;
  started_at?: string | null;
  completed_at?: string | null;
  updated_at: string;
}

export interface ApiErrorDetail {
  code: string;
  message: string;
  details?: Record<string, unknown> | null;
}

export interface ApiErrorBody {
  error: ApiErrorDetail;
}

export interface AuthResponse {
  access_token: string;
  token_type: 'Bearer' | string;
  expires_in: number;
  user: User;
}

export interface ListResponse<T> {
  items: T[];
  limit?: number;
  offset?: number;
}

export interface NotificationConsentResponse {
  notification_consent: boolean;
  notification_consent_at?: string | null;
}
