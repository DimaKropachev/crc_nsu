import { ArrowUpRight, CalendarDays, Clock3, GraduationCap, HeartHandshake, MapPin, Sparkles, Users } from 'lucide-react';
import { Button, IconTile, Notice, PageHeader } from '../shared/ui';

type StubKind = 'volunteer' | 'opportunities' | 'mailings';

const volunteerSamples = [
  { title: 'Наставничество для первокурсников', date: '15 ноября 2026', place: 'Главный корпус НГУ', description: 'Помоги новым студентам освоиться в университете и поделись своим опытом.' },
  { title: 'IT-волонтёрство на университетском хакатоне', date: '22 ноября 2026', place: 'Технопарк НГУ', description: 'Попробуй себя в организации технологического мероприятия и познакомься с командами.' },
  { title: 'Научный фестиваль для школьников', date: '5 декабря 2026', place: 'Кампус НГУ', description: 'Помоги провести интерактивные площадки и популяризировать науку.' },
];
const opportunities = [
  { category: 'СТАЖИРОВКА', title: 'Backend-разработка: летняя практика', company: 'Демонстрационная карточка', detail: 'Go · Python · SQL', description: 'Пример карточки для будущего каталога стажировок.' },
  { category: 'СТИПЕНДИЯ', title: 'Поддержка студенческих проектов', company: 'Демонстрационная карточка', detail: 'Учёба и исследования', description: 'Пример карточки для будущего раздела грантов и выплат.' },
  { category: 'КАРЬЕРА', title: 'Подготовка к первому техническому интервью', company: 'Карьерный центр НГУ', detail: 'Материалы и практика', description: 'Заготовка для будущих образовательных карьерных возможностей.' },
];

export function StubPage({ kind }: { kind: StubKind }) {
  if (kind === 'volunteer') return <VolunteerPage />;
  if (kind === 'opportunities') return <OpportunitiesPage />;
  return <MailingsPage />;
}

function VolunteerPage() {
  return <div className="page-stack"><PageHeader eyebrow="ВНЕ УЧЁБЫ — ТОЖЕ ОПЫТ" title="Волонтёрство и события" description="Участие в проектах помогает пробовать новые роли, знакомиться с людьми и собирать опыт." /><Notice tone="warning">Демонстрационный раздел. Карточки ниже — примеры интерфейса, не реальные анонсы. Регистрация на мероприятия пока не подключена.</Notice><div className="stub-feature-banner green-stub"><div><span className="eyebrow">ПОЛЕЗНЫЙ ОПЫТ</span><h2>Навыки растут там,<br />где есть практика</h2><p>Выбери активность, которая поможет попробовать себя в новой роли и дополнить портфолио.</p></div><div className="stub-feature-art"><HeartHandshake size={54} /><span>community</span></div></div><div className="section-heading compact"><div><h2>Примеры мероприятий</h2><p>Будущая лента возможностей карьерного центра</p></div><span className="count-label">Демо</span></div><div className="event-grid">{volunteerSamples.map((event, index) => <article className="event-card" key={event.title}><div className={`event-art event-art-${index + 1}`}><span>НГУ · VOLUNTEER</span><HeartHandshake size={38} /></div><div className="event-card-body"><span className="event-date"><CalendarDays size={14} /> {event.date}</span><h3>{event.title}</h3><p>{event.description}</p><span className="event-location"><MapPin size={14} /> {event.place}</span><button className="text-link disabled-link" disabled>Подробнее <ArrowUpRight size={15} /></button></div></article>)}</div></div>;
}

function OpportunitiesPage() {
  const categories = ['Все возможности', 'Стипендии', 'Стажировки', 'Карьера'];
  return <div className="page-stack"><PageHeader eyebrow="ТВОЙ СЛЕДУЮЩИЙ ШАГ" title="Стипендии и стажировки" description="Собирай подходящие возможности для профессионального развития и первого опыта." /><Notice tone="warning">Демонстрационный раздел. Здесь пока нет актуального каталога или настоящих предложений. Карточки показывают будущий формат.</Notice><div className="opportunity-hero"><div><span className="eyebrow">GROW YOUR CAREER</span><h2>Возможности не должны теряться в ленте</h2><p>В будущей версии здесь будут собраны стажировки, стипендии, конкурсы и карьерные мероприятия.</p></div><div className="opportunity-hero-icon"><GraduationCap size={58} /><Sparkles size={23} /></div></div><div className="filter-chips">{categories.map((category, index) => <button key={category} className={`filter-chip ${index === 0 ? 'active' : ''}`} disabled={index !== 0}>{category}</button>)}</div><div className="opportunity-list">{opportunities.map((item, index) => <article className="opportunity-card" key={item.title}><IconTile icon={index === 1 ? GraduationCap : index === 2 ? Sparkles : Clock3} tone={index === 1 ? 'amber' : index === 2 ? 'green' : 'blue'} size="lg" /><div className="opportunity-card-main"><span className="eyebrow">{item.category}</span><h3>{item.title}</h3><p>{item.description}</p><div className="opportunity-meta"><span>{item.company}</span><span>{item.detail}</span></div></div><button className="round-arrow disabled-arrow" aria-label="Скоро будет доступно" disabled><ArrowUpRight size={18} /></button></article>)}</div></div>;
}

function MailingsPage() {
  return <div className="page-stack"><PageHeader eyebrow="КОММУНИКАЦИЯ СО СТУДЕНТАМИ" title="Рассылки и анонсы" description="В будущем здесь можно будет готовить объявления о новых карьерных возможностях и мероприятиях." /><div className="mailing-placeholder"><div className="mailing-placeholder-icon"><Users size={31} /></div><span className="eyebrow">РАЗДЕЛ В РАЗРАБОТКЕ</span><h2>Делись возможностями со студентами</h2><p>Здесь появится редактор рассылок с выбором аудитории, предпросмотром сообщения и подтверждением отправки. Сейчас реальные сообщения не отправляются.</p><div className="mailing-steps"><span><i>01</i> Создать анонс</span><span><i>02</i> Проверить текст</span><span><i>03</i> Отправить после подтверждения</span></div><Button variant="secondary" disabled><Clock3 size={16} /> Скоро появится</Button></div><div className="mailing-note"><Sparkles size={19} /><div><strong>Пока можно сосредоточиться на консультациях</strong><p>Заявки, подтверждение встреч и расписание уже доступны в этом кабинете.</p></div></div></div>;
}
