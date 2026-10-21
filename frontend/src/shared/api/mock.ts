import type { Consultation, ConsultationSlot, Consultant, ResumeAnalysis, User } from './types';

const now = new Date();
const at = (offsetDays: number, hour: number): string => {
  const date = new Date(now);
  date.setDate(date.getDate() + offsetDays);
  date.setHours(hour, 0, 0, 0);
  return date.toISOString();
};

export const mockStudent: User = {
  id: 'a1111111-1111-4111-8111-111111111111', vk_id: 100001, first_name: 'Александр',
  last_name: 'Студент', avatar_url: null, role: 'student', notification_consent: false,
};
export const mockAdmin: User = {
  id: 'a2222222-2222-4222-8222-222222222222', vk_id: 100002, first_name: 'Мария',
  last_name: 'Консультант', avatar_url: null, role: 'admin', notification_consent: false,
};
export const mockConsultants: Consultant[] = [
  { id: 'c1111111-1111-4111-8111-111111111111', user_id: 'u1', first_name: 'Мария', last_name: 'Петрова', avatar_url: null, specialization: 'IT-карьера и развитие', description: 'Помогаю определить карьерный трек, подготовиться к техническому собеседованию и оформить опыт в резюме.', is_active: true },
  { id: 'c2222222-2222-4222-8222-222222222222', user_id: 'u2', first_name: 'Алексей', last_name: 'Смирнов', avatar_url: null, specialization: 'Резюме и собеседования', description: 'Разбираем проекты, достижения и подготовку к первому офферу.', is_active: true },
  { id: 'c3333333-3333-4333-8333-333333333333', user_id: 'u3', first_name: 'Екатерина', last_name: 'Орлова', avatar_url: null, specialization: 'Стажировки и карьера', description: 'Помогаю искать стажировки, готовиться к отбору и планировать профессиональное развитие.', is_active: true },
];
export const makeMockSlots = (consultantId: string): ConsultationSlot[] => {
  return [1, 2, 3, 4, 5, 6, 7].map((offset, index) => ({
    id: `s${consultantId.slice(1, 5)}-${offset}-4111-8111-111111111111`, consultant_id: consultantId,
    starts_at: at(offset, index % 2 ? 15 : 11), ends_at: at(offset, index % 2 ? 16 : 12), status: 'available',
  }));
};
export const mockConsultations: Consultation[] = [
  { id: 'b1111111-1111-4111-8111-111111111111', student_id: mockStudent.id, consultant_id: mockConsultants[0].id,
    slot_id: 's0001-1111-4111-8111-111111111111', starts_at: at(2, 12), ends_at: at(2, 13), problem_description: 'Подготовка к первому техническому собеседованию на Go-разработчика.',
    status: 'confirmed', reschedule_count: 0, consultant_name: 'Мария Петрова', created_at: at(-2, 10), updated_at: at(-2, 10) },
  { id: 'b2222222-2222-4222-8222-222222222222', student_id: mockStudent.id, consultant_id: mockConsultants[0].id,
    slot_id: 's0002-2222-4222-8222-222222222222', starts_at: at(5, 15), ends_at: at(5, 16), problem_description: 'Хочу улучшить резюме и понять, какие проекты стоит добавить перед поиском стажировки.',
    status: 'pending', reschedule_count: 0, consultant_name: 'Мария Петрова', created_at: at(-1, 10), updated_at: at(-1, 10) },
];
export const mockAnalyses: ResumeAnalysis[] = [];
