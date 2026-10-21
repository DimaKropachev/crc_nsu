import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { AlertCircle, CalendarDays, CheckCircle2, Clock3, FileText, LoaderCircle, type LucideIcon } from 'lucide-react';
import type { ConsultationStatus, ResumeAnalysisStatus, SlotStatus } from './api/types';

export function PageHeader({ eyebrow, title, description, action }: { eyebrow?: string; title: string; description?: string; action?: ReactNode }) {
  return <div className="page-header"><div>{eyebrow && <div className="eyebrow">{eyebrow}</div>}<h1>{title}</h1>{description && <p>{description}</p>}</div>{action && <div className="page-header-action">{action}</div>}</div>;
}

export function StatusPill({ status }: { status: ConsultationStatus | SlotStatus | ResumeAnalysisStatus }) {
  const config: Record<string, { label: string; tone: string }> = {
    pending: { label: 'Ожидает подтверждения', tone: 'warning' },
    confirmed: { label: 'Подтверждена', tone: 'success' },
    completed: { label: 'Завершена', tone: 'neutral' },
    cancelled: { label: 'Отменена', tone: 'danger' },
    available: { label: 'Свободно', tone: 'success' },
    booked: { label: 'Занято', tone: 'neutral' },
    blocked: { label: 'Заблокировано', tone: 'danger' },
    queued: { label: 'В очереди', tone: 'warning' },
    processing: { label: 'Обрабатывается', tone: 'warning' },
    failed: { label: 'Ошибка', tone: 'danger' },
  };
  const item = config[status] ?? { label: status, tone: 'neutral' };
  return <span className={`status-pill ${item.tone}`}><span className="status-dot" />{item.label}</span>;
}

export function Button({ children, variant = 'primary', size = 'md', className = '', disabled, ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary' | 'ghost' | 'danger'; size?: 'sm' | 'md' | 'lg' }) {
  return <button className={`button ${variant} ${size} ${className}`.trim()} disabled={disabled} {...props}>{children}</button>;
}

export function IconTile({ icon: Icon, tone = 'blue', size = 'md' }: { icon: LucideIcon; tone?: 'blue' | 'violet' | 'green' | 'amber' | 'pink'; size?: 'md' | 'lg' }) {
  return <span className={`icon-tile ${tone} ${size}`}><Icon size={size === 'lg' ? 23 : 20} strokeWidth={1.8} /></span>;
}

export function EmptyState({ icon: Icon = CalendarDays, title, description, action }: { icon?: LucideIcon; title: string; description: string; action?: ReactNode }) {
  return <div className="empty-state"><span className="empty-icon"><Icon size={24} /></span><h3>{title}</h3><p>{description}</p>{action}</div>;
}

export function Notice({ children, tone = 'info' }: { children: ReactNode; tone?: 'info' | 'warning' | 'danger' | 'success' }) {
  const Icon = tone === 'success' ? CheckCircle2 : tone === 'danger' ? AlertCircle : tone === 'warning' ? Clock3 : FileText;
  return <div className={`notice ${tone}`}><Icon size={18} /> <div>{children}</div></div>;
}

export function LoadingState({ label = 'Загружаем данные…' }: { label?: string }) {
  return <div className="loading-state"><LoaderCircle size={22} className="spin" /><span>{label}</span></div>;
}

export function Avatar({ name, src, size = 'md' }: { name: string; src?: string | null; size?: 'sm' | 'md' | 'lg' }) {
  const initials = name.split(' ').filter(Boolean).slice(0, 2).map((part) => part[0]?.toLocaleUpperCase('ru-RU')).join('');
  return src ? <img className={`avatar ${size}`} src={src} alt={name} /> : <span className={`avatar avatar-fallback ${size}`} aria-label={name}>{initials || 'НГУ'}</span>;
}
