// Package seed populates reference data and, optionally, a demo forum full of despair.
package seed

import (
	"errors"
	"log"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/cavelasquez17/shadowcommit/api/internal/models"
	"github.com/cavelasquez17/shadowcommit/api/internal/store"
)

// Reference upserts the data the app cannot function without: categories and badges.
// Safe to run on every boot.
func Reference(db *gorm.DB) error {
	for _, c := range categories {
		var existing models.Category
		err := db.Where("slug = ?", c.Slug).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&c).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		// Keep copy edits flowing without wiping counters.
		existing.Name, existing.Description = c.Name, c.Description
		existing.Emoji, existing.Color, existing.SortOrder = c.Emoji, c.Color, c.SortOrder
		if err := db.Save(&existing).Error; err != nil {
			return err
		}
	}

	for _, b := range badges {
		var existing models.Badge
		err := db.Where("slug = ?", b.Slug).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&b).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		existing.Label, existing.Description = b.Label, b.Description
		existing.Emoji, existing.Color = b.Emoji, b.Color
		if err := db.Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}

var categories = []models.Category{
	{Slug: "the-incident", Name: "The Incident", Emoji: "💥", Color: "#f87171", SortOrder: 1,
		Description: "Tell us exactly how it happened. Timestamps welcome. Screenshots preferred."},
	{Slug: "shadow-work", Name: "Shadow Work Confessions", Emoji: "🥷", Color: "#a78bfa", SortOrder: 2,
		Description: "Two jobs, one laptop, zero sleep. Names redacted, dignity optional."},
	{Slug: "prompt-vs-payroll", Name: "Prompt vs Payroll", Emoji: "📝", Color: "#4ade80", SortOrder: 3,
		Description: "The prompt that replaced a headcount. Post the prompt. We will judge it."},
	{Slug: "ceo-saw-a-demo", Name: "The CEO Saw A Demo", Emoji: "📺", Color: "#f472b6", SortOrder: 4,
		Description: "A LinkedIn video, a Slack message, and suddenly a roadmap."},
	{Slug: "resume-necromancy", Name: "Résumé Necromancy", Emoji: "🧟", Color: "#fbbf24", SortOrder: 5,
		Description: "Reanimating your CV. Peer review your bullet points here."},
	{Slug: "still-employed", Name: "Still Employed (Copium)", Emoji: "🫡", Color: "#22d3ee", SortOrder: 6,
		Description: "For the survivors. Post your reasons why it won't happen to you. We'll bookmark it."},
	{Slug: "show-your-repos", Name: "Show Your Repos", Emoji: "📦", Color: "#67e8f9", SortOrder: 7,
		Description: "The code that outlived your contract. Link it. Someone should see it."},
	{Slug: "hiring-and-hiding", Name: "Hiring & Hiding", Emoji: "🕳️", Color: "#86efac", SortOrder: 8,
		Description: "Real openings, real referrals, and the occasional trauma bond."},
	{Slug: "ai-did-this", Name: "AI Did This, Actually", Emoji: "🤖", Color: "#c084fc", SortOrder: 9,
		Description: "Bugs, outages and incidents shipped by a machine that never gets a PIP."},
	{Slug: "off-topic", Name: "Off Topic / Screaming", Emoji: "📣", Color: "#fdba74", SortOrder: 10,
		Description: "No agenda. Just volume."},
}

var badges = []models.Badge{
	{Slug: "shadow-exposed", Label: "Committed With My Real Name", Emoji: "🪦", Color: "#a78bfa",
		Description: "Got caught doing shadow work by git blame. A classic."},
	{Slug: "certified-obsolete", Label: "Certified Obsolete", Emoji: "📜", Color: "#f87171",
		Description: "Officially no longer required by the business."},
	{Slug: "legacy-system", Label: "Legacy System", Emoji: "🏛️", Color: "#fbbf24",
		Description: "Ten or more years of experience. Load-bearing and undocumented."},
	{Slug: "copium-enjoyer", Label: "Copium Enjoyer", Emoji: "🫡", Color: "#4ade80",
		Description: "Has Copium Mode enabled. Believes it will be fine."},
	{Slug: "polyglot-unemployed", Label: "Multilingual & Unemployed", Emoji: "🌍", Color: "#22d3ee",
		Description: "Speaks three or more languages. Rejected in all of them."},
	{Slug: "saw-the-demo", Label: "Saw The Demo", Emoji: "🛥️", Color: "#fca5a5",
		Description: "Was in the room. Made the call. Kept the equity."},
	{Slug: "founding-ghost", Label: "Founding Ghost", Emoji: "👻", Color: "#e879f9",
		Description: "Joined ShadowCommit before it was cool. Or funded."},
}

// demoUser is a seeded persona.
type demoUser struct {
	username, display, country, seniority, cause, epitaph, role, langs, bio string
	years                                                                   int
	employed                                                                bool
	repoName, repoURL, repoDesc, repoLang                                   string
}

var demoUsers = []demoUser{
	{"ghostcommit", "Andrés \"Shadow\" M.", "CO", "shadow", "shadow_work_exposed",
		"git blame killed him, not the market.", "Two full-time jobs, one identity", "es,en,pt",
		"Worked two remote jobs for 14 months. Got caught because I forgot to change the git author on one commit. My lead got fired too. Sorry, Diego.",
		9, false, "dual-standup-bot", "https://github.com/ghostcommit/dual-standup-bot",
		"Attends two standups simultaneously and paraphrases. It worked.", "TypeScript"},
	{"prompt_widow", "Yuki T.", "JP", "senior", "replaced_by_a_prompt",
		"11 words did what I did in 6 years.", "Formerly: Senior Backend", "ja,en",
		"They printed my replacement on a slide. It was a system prompt. In Comic Sans.",
		6, false, "prompt-that-replaced-me", "https://github.com/prompt_widow/prompt-that-replaced-me",
		"The literal prompt. MIT licensed. Go ahead.", "Markdown"},
	{"still_here_lol", "Priya S.", "IN", "staff", "still_alive_somehow",
		"Survives on being the only one who understands the payroll service.", "Staff Eng (load-bearing)", "hi,en,ta",
		"Job security is just undocumented code. I am fine. I am FINE.",
		11, true, "payroll-legacy", "https://github.com/still_here_lol/payroll-legacy",
		"14k lines. No tests. Nobody else can read it. This is my pension.", "COBOL"},
	{"demo_victim", "Lars B.", "DE", "cto", "ceo_saw_a_demo",
		"Showed the board a demo. The board showed him the door.", "Formerly: CTO", "de,en",
		"I presented an AI roadmap that reduced engineering headcount by 40%. Round two: the board asked what the CTO does now.",
		18, false, "roadmap-2025-final-FINAL", "https://github.com/demo_victim/roadmap-2025-final-FINAL",
		"The slide deck that ended 40 careers, including mine. Poetic.", "Keynote"},
	{"swarmfood", "Beatriz L.", "BR", "mid", "replaced_by_an_agent_swarm",
		"Took 12 agents to replace her. She considers this a promotion.", "Between opportunities", "pt,es,en",
		"They needed a twelve-agent orchestration with a human reviewer to do my job. The human reviewer is now also gone.",
		5, false, "agent-swarm-audit", "https://github.com/swarmfood/agent-swarm-audit",
		"I audited the swarm that replaced me. It hallucinates 31% of migrations.", "Python"},
	{"exit_first", "Kwame A.", "GH", "lead", "left_before_they_could_fire_me",
		"Undefeated. Resigned on a Tuesday, layoffs landed Wednesday.", "Consultant (by force)", "en,fr,tw",
		"I read the tea leaves in the Q3 OKRs. Resigned 18 hours before the announcement. Best commit of my career.",
		12, false, "okr-tea-leaves", "https://github.com/exit_first/okr-tea-leaves",
		"Parses your company OKRs and estimates your remaining tenure.", "Go"},
	{"copilot_ate_me", "Marta K.", "PL", "junior", "copilot_did_it_faster",
		"Was learning. The machine had already learned.", "Job seeking, aggressively", "pl,en,de",
		"I was 4 months into my first job. My mentor started pair-programming with Copilot instead of me. I understood.",
		1, false, "junior-tasks-archive", "https://github.com/copilot_ate_me/junior-tasks-archive",
		"Every ticket a junior used to get. Now closed by a bot in 40 seconds.", "JavaScript"},
	{"synergy_survivor", "Diego R.", "AR", "principal", "synergistic_restructuring",
		"Reorganised into a shape that did not include him.", "Principal (of nothing)", "es,en",
		"HR called it a 'shape change'. I asked what shape. They said 'flatter'. I was the height.",
		15, false, "orgchart-diff", "https://github.com/synergy_survivor/orgchart-diff",
		"Diffs your org chart weekly and alerts you when your box loses an edge.", "Rust"},
}

// demoThread is a seeded thread with its replies.
type demoThread struct {
	authorIdx int
	category  string
	title     string
	body      string
	lang      string
	sarcasm   int
	tags      []string
	replies   []demoReply
}

type demoReply struct {
	authorIdx int
	body      string
	lang      string
}

var demoThreads = []demoThread{
	{0, "shadow-work", "I did two jobs for 14 months and one commit ended both",
		`Fourteen months. Two full-time remote roles. Two laptops, two VPNs, two Slack workspaces with different notification sounds so I could tell them apart in my sleep.

I had a whole system. Separate git configs per directory. A shell alias that refused to commit if the author email did not match the repo remote. It worked flawlessly.

Then I fixed a "quick" prod bug at 2am from the wrong terminal.

` + "```" + `
commit 8f2a1c9
Author: Andrés M. <andres@company-A.com>
Date:   Tue 02:14

    hotfix: null check on invoice total
` + "```" + `

That commit is in **Company B's** repository.

Twelve hours later I am in a call with two people I have never met. Forty minutes after that, my lead is in the call. He did not know. He genuinely did not know. They fired him anyway, for "insufficient oversight".

Diego, if you find this account: I'm sorry. Your reference letter is the best thing I have ever written and you will never read it.

The AI part? They replaced *both* of us with a Claude subscription and a junior. The junior lasted five weeks.`,
		"en", 95, []string{"shadow work", "git blame", "regret"},
		[]demoReply{
			{3, "The `git config user.email` per-directory trick is the load-bearing wall of the entire shadow economy and you drilled through it at 2am. Genuinely devastating. 💀", "en"},
			{7, "Hermano, lo mismo me pasó pero con un archivo `.env` que subí al repo equivocado. La diferencia es que el mío tenía las credenciales de producción. Me fui con más ruido.", "es"},
			{2, "The lead getting fired is the part that should be illegal. He had no visibility, no tooling, and no chance. The org failed at oversight and charged him for it.", "en"},
			{5, "Undefeated take: you got 14 months of double salary. Amortised over one commit, that commit was worth about $180k. Best line of code anyone here has written.", "en"},
		}},
	{1, "prompt-vs-payroll", "Me reemplazaron con 11 palabras. Aquí están, gratis",
		`Seis años. Arquitectura de microservicios, migraciones de datos, on-call rotativo, tres incidentes de sev1 resueltos a las 4am.

Me reemplazaron con esto:

> "Refactoriza este servicio siguiendo los patrones existentes y añade tests."

Once palabras. Las conté. Las conté como cuarenta veces.

Lo peor no es que funcione. Funciona regular, honestamente — el 30% de las veces inventa un import que no existe. Lo peor es que **funciona lo suficiente**. Y "lo suficiente" es más barato que yo.

Mi ex-manager me dijo, textual: "no es personal, es matemática". Y tiene razón, que es exactamente lo que lo hace insoportable.

Subí el prompt a GitHub con licencia MIT. Si te va a reemplazar, que al menos sea open source.

github.com/prompt_widow/prompt-that-replaced-me ⭐`,
		"es", 100, []string{"prompt", "matematica", "mit license"},
		[]demoReply{
			{4, "Licenciarlo con MIT es el nivel de petty que aspiro a alcanzar. Deberías añadir un `CONTRIBUTING.md` que diga 'no'.", "es"},
			{0, "The \"it works well enough\" line is the whole industry in five words. Nobody was ever fired for quality. They were fired for margin.", "en"},
			{6, "Ich habe deinen Prompt getestet. Er erzeugt bei mir einen Import aus einem Paket, das seit 2019 nicht existiert. Du wurdest von einem Halluzinierenden ersetzt. Trage es mit Stolz.", "de"},
			{2, "Counterpoint from someone still employed: those 11 words don't include the 6 years of context you had about WHY the patterns exist. They'll find out. Around month nine.", "en"},
		}},
	{3, "ceo-saw-a-demo", "I made the deck that fired 40 people. Slide 12 fired me.",
		`I was the CTO. I want to be clear about that, because I am not asking for sympathy — I am filing a confession.

Board meeting, Q4 planning. I presented "AI-Native Engineering Org", a 34-slide deck arguing we could ship the same roadmap with 60% of engineering. I believed it. I had benchmarks. I had a proof of concept that closed 200 tickets in a weekend.

The board loved it. Approved in eleven minutes. Forty roles gone by February.

Slide 12 was titled "Reduced Coordination Overhead". It had a diagram showing fewer management layers.

At the March board meeting a director pulled up slide 12 and asked, and I remember the exact phrasing: *"So if engineering coordinates itself now — what does the CTO do?"*

I did not have a slide for that.

I built the argument for my own redundancy, in Keynote, with my own animations. The transitions were beautiful.

The deck is on GitHub. It's not code. I don't care.`,
		"en", 90, []string{"ceo", "slide 12", "keynote"},
		[]demoReply{
			{7, "\"I did not have a slide for that\" is going to live in my head rent free until the machines take that too. Genuinely the best thing posted on this site.", "en"},
			{4, "Slide 12 fez um trabalho melhor que qualquer sindicato. Você automatizou a própria demissão com transições animadas. Isso é arte performática.", "pt"},
			{2, "Respect for posting this under your real seniority instead of hiding as \"former tech leader\". The 40 people won't forgive you but the honesty counts for something.", "en"},
			{0, "You and I are the same story from opposite ends. I got fired for one commit. You got fired for one slide. The medium doesn't matter, only the artifact.", "en"},
		}},
	{4, "ai-did-this", "O swarm que me substituiu derrubou a produção. Eu fui chamado de volta. Como consultor. Por 3x.",
		`Fui demitida em janeiro. Substituída por uma orquestração de doze agentes com um revisor humano.

Em março o revisor humano também foi demitido. Otimização de custos.

Em abril, às 3h da manhã de um domingo, o swarm executou uma migração que ele mesmo tinha gerado. A migração renomeou uma coluna em produção. A migração **não** atualizou os 47 lugares que liam aquela coluna.

Downtime: 9 horas. Receita perdida: eles não me disseram, mas o CFO chorou numa call, o que é um número.

Meu ex-diretor me ligou no domingo às 11h. Não perguntou como eu estava. Perguntou se eu "tinha disponibilidade".

Cobrei três vezes minha antiga taxa. Ele aceitou em quatro segundos. Quatro. Eu contei.

Auditei o swarm depois. Taxa de alucinação em migrações: **31%**. Eles rodaram isso em produção sem revisor por seis semanas. Foi menos sorte do que estatística.

Repo do audit no meu perfil. É Python, é feio, funciona.`,
		"pt", 100, []string{"swarm", "producao", "consultoria"},
		[]demoReply{
			{2, "31% hallucination rate on migrations, running unsupervised in prod for six weeks. That's not a bug, that's a company that decided the outage was cheaper than your salary. And they were wrong.", "en"},
			{5, "Charging 3x and having him accept in four seconds tells you exactly what your salary should have been. You were never expensive. You were underpriced.", "en"},
			{1, "彼が土曜の夜ではなく日曜の朝11時に電話してきたのが最高です。9時間ダウンしてから、ようやくプライドを飲み込んだんですね。", "ja"},
			{6, "Jako junior, którego wywalili po 4 miesiącach: to jest dokładnie ta przyszłość, o której mi mówili, że nie nastąpi. Dziękuję za dowody.", "pl"},
		}},
	{2, "still-employed", "Reasons I personally will never be replaced (bookmark this, I'll be back)",
		`I am posting this in good faith and full awareness that this thread is a time capsule and I am the punchline.

**Why I am safe:**

1. I maintain the payroll service. 14,000 lines of COBOL, zero tests, written by a man who retired in 2011 and died in 2016.
2. There is no documentation. There is a Word document called "payroll_notes_FINAL_v3.doc" that describes a system we decommissioned.
3. The build depends on a specific version of a compiler that is no longer distributed. It lives on a VM. The VM has a name. Its name is Gerald.
4. If payroll breaks, nobody gets paid, including the executives who would like to fire me.
5. I have fed sections of this codebase to every model available. Every single one confidently rewrites it in a way that silently breaks the tax withholding logic for one specific edge case. Then it tells me the refactor is complete.

Job security is not skill. Job security is **being the only load-bearing element in a structure nobody wants to inspect.**

Ask me again in eighteen months.`,
		"en", 85, []string{"cobol", "gerald", "job security"},
		[]demoReply{
			{0, "Naming the VM Gerald is the single strongest career move documented on this website. When Gerald dies, you die. Please back up Gerald.", "en"},
			{3, "As the guy who made the deck: your #4 is the only argument that ever worked in a board room. Not \"this is hard\", but \"this failing costs you personally\". Everything else is negotiable.", "en"},
			{7, "Punto 5 es la parte que la gente no entiende. No es que no pueda reescribirlo. Es que reescribe con confianza total y rompe UN caso fiscal que nadie va a notar hasta abril.", "es"},
			{4, "Colocando um alarme para daqui a 18 meses. Sem ressentimentos, é só ciência.", "pt"},
		}},
	{6, "the-incident", "4 months into my first job. My mentor started pairing with Copilot instead of me.",
		`I don't have a dramatic story. That's kind of the point.

Nobody fired me in a call. Nobody read me a script. What happened is that the pairing sessions just... stopped getting scheduled.

Week 1-8: two sessions a week with Tomasz. He was genuinely great. Explained things twice without making it weird.

Week 9: one session.

Week 12: I asked. He said "honestly, I've been prototyping with Copilot, it's faster for the first pass". Not unkind. Just true.

Week 14: my tickets started closing before standup. Not by me.

Week 16: "we're restructuring the graduate program".

I was not replaced by an AI. I was replaced by the *absence of a reason to train me*. There's no ticket for teaching a junior. There's no OKR. It was always charity funded by slack in the schedule, and the slack is gone.

I'm not angry at Tomasz. I'd have done the same. That's the worst part — I understand the decision completely and it still ended my career before it started.

I'm 23. I don't know what to do with a CV that says "4 months".`,
		"en", 60, []string{"junior", "graduate program", "mentorship"},
		[]demoReply{
			{2, "This is the most important thread on this site and it has the fewest jokes. The pipeline isn't being cut, it's being quietly defunded, and in six years there will be no seniors either. Nobody is planning for that.", "en"},
			{0, "Put the 4 months on the CV and put the reason next to it. \"Graduate program discontinued, Q1 2026\" is a fact about them, not about you. Any hiring manager under 40 knows exactly what happened.", "en"},
			{5, "I'm hiring, actually. Small shop, boring domain, we still pair with humans because our compliance officer won't let the models near the data. DM me. This is not charity, juniors are cheap and I am cheap.", "en"},
			{1, "23歳。私は29歳で同じ場所にいます。あなたには時間があります。私にはローンがあります。", "ja"},
			{7, "Lo de \"no fui reemplazado por una IA, fui reemplazado por la ausencia de una razón para entrenarme\" debería estar en la portada de este sitio.", "es"},
		}},
	{5, "resume-necromancy", "Rate my CV bullet: \"Reduced team headcount dependency by 100%\"",
		`Currently rewriting my CV and I want peer review on the framing, because the honest version is unhireable.

**Honest version:** "Read the Q3 OKRs, correctly predicted layoffs, resigned 18 hours before the announcement."

**LinkedIn version:** "Proactively identified organisational risk signals and executed a strategic transition ahead of a company-wide restructuring."

**Version I actually want to use:** "Undefeated."

Other bullets under consideration:

- "Delivered 12 years of institutional knowledge to a competitor at zero cost to myself" ✅ true
- "Reduced team headcount dependency by 100%" ⚠️ technically about my own departure
- "Pioneered async standups" 🚩 I just stopped attending them

Recruiters, be honest: does "Undefeated" get me a screen or a block?`,
		"en", 100, []string{"cv", "linkedin", "undefeated"},
		[]demoReply{
			{3, "Ex-CTO here, I read maybe 400 CVs. \"Undefeated\" gets you a screen from anyone who has been fired and a block from anyone who hasn't. In 2026 that's favourable odds.", "en"},
			{2, "\"Pioneered async standups\" is a lie so beautiful I'm stealing it. I also just stopped attending them. Nobody noticed for five weeks.", "en"},
			{4, "A versão do LinkedIn é boa mas falta um substantivo abstrato. Tenta \"executed a strategic transition ahead of a company-wide organizational realignment initiative\". Agora sim parece que você não fez nada.", "pt"},
			{6, "Napisz \"Undefeated\". Ja napisałam \"4 months\" i nikt nie odpowiedział. Odwaga jest tańsza niż uczciwość.", "pl"},
		}},
	{7, "shadow-work", "HR llamó a mi despido un \"cambio de forma\". Yo era la altura.",
		`Quince años. Principal Engineer. Tres reorganizaciones sobrevividas.

La cuarta fue por videollamada, un martes, 14 minutos. La persona de RRHH usó la frase "estamos cambiando la forma del equipo".

Pregunté qué forma.

Dijo: "más plana".

Silencio de tres segundos. Yo era la parte alta. Ese era el chiste y ninguno de los dos lo dijo en voz alta.

Lo que reemplazó mi función: un agente que resume decisiones técnicas y las publica en Confluence. Literalmente eso era el 40% de mi trabajo y lo sabía. La otra parte — decir "no" a arquitecturas malas con autoridad suficiente para que se detengan — eso no lo reemplazaron. Simplemente dejaron de tener a nadie que lo hiciera.

Escribí una herramienta que hace diff de tu organigrama semanalmente y te avisa cuando tu caja pierde una conexión. Se llama "orgchart-diff". Está en Rust porque estaba enojado.

Ojalá la hubiera escrito en 2023.`,
		"es", 95, []string{"rrhh", "organigrama", "rust"},
		[]demoReply{
			{0, "\"Está en Rust porque estaba enojado\" es la mejor justificación de stack tecnológico jamás escrita. Voy a usar esa herramienta hoy mismo.", "es"},
			{2, "The part nobody's pricing in: they replaced your documentation duties and kept the vacuum where \"authority to say no\" used to be. That vacuum has a cost. It arrives in about four quarters, as an outage.", "en"},
			{3, "\"More flat\" is the exact phrase from slide 12 of my deck. I want you to know that the language is standardised. There is a template. We are all reading from the same file.", "en"},
			{5, "orgchart-diff instalado. Ya me avisó de dos cosas. No sé si agradecerte o bloquearte.", "es"},
		}},
}

// Demo populates the forum with seeded personas and threads. It is a no-op if users exist.
func Demo(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		log.Println("seed: users already exist, skipping demo content")
		return nil
	}

	rng := rand.New(rand.NewSource(1337))
	now := time.Now()

	users := make([]models.User, 0, len(demoUsers))
	for i, du := range demoUsers {
		u := models.User{
			PublicID:        store.NewPublicID("usr"),
			Username:        du.username,
			DisplayName:     du.display,
			AvatarURL:       "https://api.dicebear.com/9.x/bottts-neutral/svg?seed=" + du.username,
			Bio:             du.bio,
			CountryCode:     du.country,
			Seniority:       models.Seniority(du.seniority),
			YearsExp:        du.years,
			CauseOfDeath:    models.CauseOfDeath(du.cause),
			Epitaph:         du.epitaph,
			CurrentRole:     du.role,
			IsEmployed:      du.employed,
			GitHubURL:       "https://github.com/" + du.username,
			UILanguage:      strings.Split(du.langs, ",")[0],
			SpokenLanguages: du.langs,
			CopiumMode:      du.cause == "still_alive_somehow",
			Role:            "member",
			CreatedAt:       now.Add(-time.Duration(60-i*5) * 24 * time.Hour),
		}
		if !du.employed {
			fired := now.Add(-time.Duration(rng.Intn(200)+10) * 24 * time.Hour)
			u.FiredAt = &fired
		}
		if err := db.Create(&u).Error; err != nil {
			return err
		}
		if err := db.Create(&models.Repo{
			PublicID: store.NewPublicID("rep"), UserID: u.ID,
			Name: du.repoName, URL: du.repoURL, Description: du.repoDesc,
			Language: du.repoLang, Stars: rng.Intn(900) + 12,
			WrittenByAI: rng.Intn(3) == 0,
		}).Error; err != nil {
			return err
		}
		users = append(users, u)
	}

	// Everyone gets the founding badge; the rest are earned through profile state.
	var founding models.Badge
	if err := db.Where("slug = ?", "founding-ghost").First(&founding).Error; err == nil {
		for i := range users {
			_ = db.Model(&users[i]).Association("Badges").Append(&founding)
		}
	}

	cats := map[string]models.Category{}
	var allCats []models.Category
	if err := db.Find(&allCats).Error; err != nil {
		return err
	}
	for _, c := range allCats {
		cats[c.Slug] = c
	}

	kinds := make([]models.ReactionKind, 0, len(reactionKinds))
	kinds = append(kinds, reactionKinds...)

	for ti, dt := range demoThreads {
		cat, ok := cats[dt.category]
		if !ok {
			continue
		}
		created := now.Add(-time.Duration(len(demoThreads)-ti) * 19 * time.Hour)
		t := models.Thread{
			PublicID: store.NewPublicID("thr"), AuthorID: users[dt.authorIdx].ID,
			CategoryID: cat.ID, Title: dt.title, Slug: slugify(dt.title),
			Body: dt.body, Language: dt.lang, Sarcasm: dt.sarcasm,
			ViewCount: rng.Intn(4000) + 120, CreatedAt: created, UpdatedAt: created,
		}
		if err := db.Create(&t).Error; err != nil {
			return err
		}

		for _, label := range dt.tags {
			slug := slugify(label)
			var tag models.Tag
			if err := db.Where("slug = ?", slug).First(&tag).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				tag = models.Tag{Slug: slug, Label: label, UsageCount: 1}
				if err := db.Create(&tag).Error; err != nil {
					return err
				}
			} else {
				db.Model(&tag).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
			}
			_ = db.Model(&t).Association("Tags").Append(&tag)
		}

		lastAt := created
		for ri, dr := range dt.replies {
			postedAt := created.Add(time.Duration(ri+1) * 97 * time.Minute)
			p := models.Post{
				PublicID: store.NewPublicID("pst"), ThreadID: t.ID,
				AuthorID: users[dr.authorIdx].ID, Body: dr.body, Language: dr.lang,
				CreatedAt: postedAt, UpdatedAt: postedAt,
			}
			if err := db.Create(&p).Error; err != nil {
				return err
			}
			lastAt = postedAt

			// Sprinkle reactions on replies.
			for _, ui := range rng.Perm(len(users))[:rng.Intn(4)+1] {
				if users[ui].ID == p.AuthorID {
					continue
				}
				_ = db.Create(&models.Reaction{
					UserID: users[ui].ID, Kind: kinds[rng.Intn(len(kinds))],
					TargetType: "post", TargetID: p.ID, CreatedAt: postedAt,
				}).Error
			}
			var rc int64
			db.Model(&models.Reaction{}).Where("target_type = ? AND target_id = ?", "post", p.ID).Count(&rc)
			db.Model(&p).UpdateColumn("reaction_count", rc)
		}

		// And a healthy pile on the thread itself.
		for _, ui := range rng.Perm(len(users)) {
			if users[ui].ID == t.AuthorID {
				continue
			}
			for _, k := range kinds {
				if rng.Intn(3) != 0 {
					continue
				}
				_ = db.Create(&models.Reaction{
					UserID: users[ui].ID, Kind: k,
					TargetType: "thread", TargetID: t.ID, CreatedAt: created,
				}).Error
			}
		}
		var trc int64
		db.Model(&models.Reaction{}).Where("target_type = ? AND target_id = ?", "thread", t.ID).Count(&trc)
		db.Model(&t).Updates(map[string]any{
			"reaction_count": trc,
			"post_count":     len(dt.replies),
			"last_post_at":   lastAt,
			"hot_score":      float64(trc+1) / pow15(time.Since(created).Hours()+2),
		})
		db.Model(&models.Category{}).Where("id = ?", cat.ID).
			UpdateColumn("thread_count", gorm.Expr("thread_count + 1"))
	}

	// Recompute the denormalised per-user counters from the truth.
	for i := range users {
		var tc, pc, rep int64
		db.Model(&models.Thread{}).Where("author_id = ?", users[i].ID).Count(&tc)
		db.Model(&models.Post{}).Where("author_id = ?", users[i].ID).Count(&pc)
		db.Raw(`SELECT COUNT(*) FROM reactions r
		        WHERE (r.target_type = 'thread' AND r.target_id IN (SELECT id FROM threads WHERE author_id = ?))
		           OR (r.target_type = 'post'   AND r.target_id IN (SELECT id FROM posts   WHERE author_id = ?))`,
			users[i].ID, users[i].ID).Scan(&rep)
		db.Model(&users[i]).Updates(map[string]any{
			"thread_count": tc, "post_count": pc, "reputation": rep,
		})
	}

	// A small follow graph so the social features have something to render.
	for i := range users {
		for _, j := range rng.Perm(len(users))[:3] {
			if i == j {
				continue
			}
			_ = db.Create(&models.Follow{FollowerID: users[i].ID, FollowingID: users[j].ID}).Error
		}
	}

	// Seed a few job postings so "Hire the Fallen" isn't empty on first load.
	demoJobs := []models.JobPosting{
		{
			PublicID: store.NewPublicID("job"), PostedByID: users[0].ID,
			Company: "Acme Corp", Title: "Senior Backend Engineer (No AI Co-Pilot Required)",
			Body: "We need someone who can write Go without asking a chatbot. You will own the API layer, the database schema, and the on-call rotation. We tried replacing our last engineer with an agent swarm. It filed three Jira tickets and took a nap.",
			ApplyURL: "https://example.com/apply/acme", Location: "Remote", Remote: true,
			SalaryMin: ptrInt(120000), SalaryMax: ptrInt(160000), Currency: "USD",
			Seniority: models.SenioritySenior, Language: "en", IsSponsored: true,
		},
		{
			PublicID: store.NewPublicID("job"), PostedByID: users[1].ID,
			Company: "StartupCo", Title: "Full-Stack Developer — Humans Only",
			Body: "React + Node. We don't care if you use Copilot, but you have to understand what it generates. If you can debug your own code, you're already overqualified for our last hire.",
			ApplyURL: "https://example.com/apply/startupco", Location: "São Paulo, BR", Remote: true,
			CountryCode: "BR", SalaryMin: ptrInt(80000), SalaryMax: ptrInt(120000), Currency: "USD",
			Seniority: models.SeniorityMid, Language: "pt",
		},
		{
			PublicID: store.NewPublicID("job"), PostedByID: users[2].ID,
			Company: "Legacy Systems Inc", Title: "Principal Engineer — Load-Bearing Human",
			Body: "We have a system that has been in production since 2008. Nobody knows how it works. The last person who did was replaced by a prompt. We need someone who can read code older than the LLM that tried to replace them.",
			ApplyURL: "https://example.com/apply/legacy", Location: "Berlin, DE", Remote: false,
			CountryCode: "DE", SalaryMin: ptrInt(150000), SalaryMax: ptrInt(200000), Currency: "EUR",
			Seniority: models.SeniorityPrincipal, Language: "en", IsSponsored: true,
		},
	}
	for i := range demoJobs {
		if err := db.Create(&demoJobs[i]).Error; err != nil {
			return err
		}
	}

	log.Printf("seed: created %d users and %d threads full of documented despair", len(users), len(demoThreads))
	return nil
}

var reactionKinds = []models.ReactionKind{
	models.ReactionF, models.ReactionRelate, models.ReactionLaugh,
	models.ReactionRobot, models.ReactionCopium, models.ReactionBrutal, models.ReactionLawyer,
}

func pow15(x float64) float64 { return x * sqrt(x) }

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for range 24 {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// slugify is duplicated here rather than exported from handlers: the seed package must not
// depend on the HTTP layer.
func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	out = strings.Trim(out, "-")
	if len(out) > 80 {
		out = strings.Trim(out[:80], "-")
	}
	return out
}

func ptrInt(v int) *int { return &v }
