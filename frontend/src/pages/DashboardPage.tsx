import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { ArrowRight, ArrowUpRight, CalendarDays, BellRing, CheckCircle2, ChevronRight, Clock3, FileText, Sparkles, Users, Video } from 'lucide-react';
import { useAuth } from '../app/AuthProvider';
import { adminApi, authApi, consultationsApi, consultantsApi } from '../shared/api/api';
import { Avatar, Button, EmptyState, IconTile, LoadingState, Notice, PageHeader, StatusPill } from '../shared/ui';
import { formatDateTime } from '../shared/lib/dates';

export function DashboardPage() {
  const { user } = useAuth();
  return user?.role === 'admin' ? <AdminDashboard /> : <StudentDashboard />;
}

function StudentDashboard() {
  const navigate = useNavigate();
  const { user, updateNotificationConsent } = useAuth();
  const queryClient = useQueryClient();
  const consent = useMutation({ mutationFn: (enabled: boolean) => authApi.consent(enabled), onSuccess: async (response) => { updateNotificationConsent(response.notification_consent); await queryClient.invalidateQueries({ queryKey: ['consultations', 'mine'] }); } });
  const consultations = useQuery({ queryKey: ['consultations', 'mine'], queryFn: consultationsApi.mine });
  const consultants = useQuery({ queryKey: ['consultants'], queryFn: consultantsApi.list });
  const upcoming = (consultations.data ?? []).filter((item) => ['pending', 'confirmed'].includes(item.status) && new Date(item.starts_at) > new Date()).sort((a, b) => a.starts_at.localeCompare(b.starts_at));
  const firstName = user?.first_name || 'студент';

  return <div className="page-stack dashboard-page">
    <section className="welcome-hero">
      <div className="hero-content">
        <h1>
          Привет, {firstName}!<br />
          <span>Давай двигаться к цели.</span>
        </h1>
        <button className="button primary btn-hero" onClick={() => navigate('/booking')}>
          Записаться на консультацию
          <ArrowRight size={18} />
        </button>
      </div>
      
      <div className="hero-logo-container">
        <img src={`${import.meta.env.BASE_URL}logo.png`} alt="НГУ" className="hero-logo-img" />
        <div className="hero-logo-text">
          <span className="hero-logo-title">Центр развития<br/>карьеры</span>
          <span className="hero-logo-sub">НГУ</span>
        </div>
      </div>
    </section>

    <div className="section-heading"><div><h2>Твои инструменты</h2></div></div>
    <div className="feature-grid">
      <button className="feature-card featured-blue" onClick={() => navigate('/booking')}><span className="feature-card-top"><IconTile icon={CalendarDays} tone="blue" size="lg" /><span className="feature-arrow"><ArrowUpRight size={19} /></span></span><span className="feature-title">Карьерная консультация</span><span className="feature-desc">Обсуди карьерную цель, подготовься к интервью или выбери направление развития.</span><span className="feature-bottom"><span>Выбрать консультанта</span><ChevronRight size={16} /></span></button>
      <button className="feature-card featured-violet" onClick={() => navigate('/resume')}><span className="feature-card-top"><IconTile icon={Sparkles} tone="violet" size="lg" /><span className="feature-arrow"><ArrowUpRight size={19} /></span></span><span className="feature-title">Анализ резюме</span><span className="feature-desc">Получи конкретные рекомендации и сделай резюме сильнее.</span><span className="feature-bottom"><span>Проверить резюме</span><ChevronRight size={16} /></span></button>
      <button type="button" className="feature-card is-disabled" disabled aria-disabled="true"><span className="wip-badge">В разработке</span><span className="feature-card-top"><IconTile icon={Users} tone="green" size="lg" /><span className="feature-arrow"><ArrowUpRight size={19} /></span></span><span className="feature-title">Волонтёрство</span><span className="feature-desc">Найди проект, попробуй себя в новой роли и получи полезный опыт.</span><span className="feature-bottom"><span>Скоро откроем</span><ChevronRight size={16} /></span></button>
      <button type="button" className="feature-card is-disabled" disabled aria-disabled="true"><span className="wip-badge">В разработке</span><span className="feature-card-top"><IconTile icon={FileText} tone="amber" size="lg" /><span className="feature-arrow"><ArrowUpRight size={19} /></span></span><span className="feature-title">Стипендии и стажировки</span><span className="feature-desc">Возможности для учёбы, практики и первых шагов в профессии.</span><span className="feature-bottom"><span>Скоро откроем</span><ChevronRight size={16} /></span></button>
    </div>

    <div className="dashboard-columns">
      <section className="panel-card upcoming-panel"><div className="panel-heading"><div><h2>Ближайшие консультации</h2></div><button className="text-link" onClick={() => navigate('/consultations')}>Все записи <ChevronRight size={16} /></button></div>
        {consultations.isLoading ? <LoadingState /> : consultations.isError ? <Notice tone="danger">Не удалось загрузить консультации. Обнови страницу или попробуй позже.</Notice> : upcoming.length === 0 ? <EmptyState icon={CalendarDays} title="Пока нет встреч" description="Выбери консультанта и запланируй первую встречу." action={<Button variant="secondary" onClick={() => navigate('/booking')}>Выбрать время</Button>} /> : <div className="upcoming-list">{upcoming.slice(0, 2).map((item) => <div className="upcoming-item" key={item.id}><div className="date-block"><strong>{new Date(item.starts_at).getDate()}</strong><span>{new Intl.DateTimeFormat('ru-RU', { month: 'short' }).format(new Date(item.starts_at)).replace('.', '')}</span></div><div className="upcoming-main"><strong>{item.consultant_name || 'Карьерный консультант'}</strong><span>{formatDateTime(item.starts_at)}</span><span className="muted-line">{item.problem_description || 'Карьерная консультация'}</span></div><StatusPill status={item.status} /></div>)}</div>}
      </section>
      <section className="panel-card consultant-panel"><div className="panel-heading"><div><h2>Наши консультанты</h2></div></div>{consultants.isLoading ? <LoadingState /> : consultants.isError ? <Notice tone="danger">Не удалось загрузить список консультантов.</Notice> : <div className="consultant-preview-list">{(consultants.data ?? []).slice(0, 3).map((consultant) => <div className="consultant-preview" key={consultant.id}><Avatar name={`${consultant.first_name} ${consultant.last_name}`} src={consultant.avatar_url} size="md" /><div><strong>{consultant.first_name} {consultant.last_name}</strong><span>{consultant.specialization}</span></div><button className="round-arrow" aria-label={`Выбрать ${consultant.first_name}`} onClick={() => navigate('/booking')}><ArrowUpRight size={17} /></button></div>)}</div>}
      <button className="panel-bottom-link" onClick={() => navigate('/booking')}>Выбрать консультанта <ArrowUpRight size={15} /></button></section>
    </div>
    <section className="notification-consent-card"><span className="notification-icon"><BellRing size={20} /></span><div className="notification-consent-copy"><strong>Напоминания о консультациях</strong><p>Разреши сервису подготовить напоминание за 24 часа до встречи через VK. Доставка зависит от настроек и разрешений VK на сервере.</p>{consent.isError && <span className="field-error">Не удалось сохранить настройку. Попробуй ещё раз.</span>}</div><Button variant={user?.notification_consent ? 'secondary' : 'primary'} disabled={consent.isPending} onClick={() => consent.mutate(!user?.notification_consent)}>{consent.isPending ? 'Сохраняем…' : user?.notification_consent ? 'Выключить' : 'Включить'}</Button></section>
  </div>;
}

function AdminDashboard() {
  const navigate = useNavigate();
  const consultations = useQuery({ queryKey: ['admin-consultations'], queryFn: () => adminApi.consultations() });
  const records = consultations.data ?? [];
  const pending = records.filter((item) => item.status === 'pending');
  const confirmed = records.filter((item) => item.status === 'confirmed');
  return <div className="page-stack">
    <PageHeader title="Рабочий кабинет" action={<Button onClick={() => navigate('/admin')}><CalendarDays size={17} /> Открыть расписание</Button>} />
    <div className="stat-grid"><div className="stat-card"><div className="stat-label">Всего записей</div><div className="stat-value">{consultations.isLoading ? '—' : records.length}</div><div className="stat-foot"><CalendarDays size={15} /> Все консультации</div></div><div className="stat-card"><div className="stat-label">Нужно подтвердить</div><div className="stat-value">{consultations.isLoading ? '—' : pending.length}</div><div className="stat-foot amber-text"><Clock3 size={15} /> Ожидают ответа</div></div><div className="stat-card"><div className="stat-label">Подтверждены</div><div className="stat-value">{consultations.isLoading ? '—' : confirmed.length}</div><div className="stat-foot green-text"><CheckCircle2 size={15} /> Запланированные встречи</div></div></div>
    {consultations.isError && <Notice tone="danger">Не удалось загрузить записи консультанта.</Notice>}
    <section className="panel-card"><div className="panel-heading"><div><h2>Требуют внимания</h2><p>Новые заявки студентов</p></div><button className="text-link" onClick={() => navigate('/admin')}>Все консультации <ChevronRight size={16} /></button></div>{consultations.isLoading ? <LoadingState /> : pending.length === 0 ? <EmptyState icon={CheckCircle2} title="Все заявки обработаны" description="Когда появятся новые записи, они отобразятся здесь." /> : <div className="upcoming-list">{pending.slice(0, 4).map((item) => <div className="upcoming-item" key={item.id}><div className="date-block"><strong>{new Date(item.starts_at).getDate()}</strong><span>{new Intl.DateTimeFormat('ru-RU', { month: 'short' }).format(new Date(item.starts_at)).replace('.', '')}</span></div><div className="upcoming-main"><strong>{item.student_name || 'Студент НГУ'}</strong><span>{formatDateTime(item.starts_at)}</span><span className="muted-line">{item.problem_description || 'Карьерная консультация'}</span></div><StatusPill status={item.status} /></div>)}</div>}</section>
    <div className="feature-grid admin-shortcuts"><button className="feature-card" onClick={() => navigate('/admin')}><IconTile icon={CalendarDays} tone="blue" /><span className="feature-title">Расписание</span><span className="feature-desc">Создавайте свободные слоты и управляйте доступностью времени.</span><span className="feature-bottom">Открыть раздел <ChevronRight size={16} /></span></button><div className="feature-card is-disabled"><span className="wip-badge">В разработке</span><IconTile icon={Video} tone="violet" /><span className="feature-title">Новости и рассылки</span><span className="feature-desc">Раздел для будущих анонсов мероприятий, стажировок и стипендий.</span><span className="feature-bottom">Посмотреть раздел <ChevronRight size={16} /></span></div></div>
  </div>;
}
