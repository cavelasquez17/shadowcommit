/**
 * Copium filter — the AI that replaced you now "enhances" your complaints.
 *
 * When Copium mode is on, all user-generated text gets passed through a
 * dictionary of corporate euphemisms so absurdly positive that the
 * contrast with reality IS the joke. "Fired" becomes "liberated to pursue
 * synergistic growth opportunities in a dynamic talent ecosystem".
 *
 * The filter is client-side, reversible, and deliberately insulting —
 * the sanitization is as transparent as the layoff was inevitable.
 */

// Ordered replacements — longer phrases first so they match before substrings.
const COPIUM_MAP: [RegExp, string][] = [
  // ═══════════════════════════════════════════════════════════════
  // ENGLISH — HR-speak so toxic-positive it circles back to offensive
  // ═══════════════════════════════════════════════════════════════

  // Phrases (longest first)
  [/\bI was fired\b/gi, 'I was liberated to pursue my passion for not being here'],
  [/\bthey fired me\b/gi, 'they kindly empowered me to explore external growth opportunities'],
  [/\bfired me\b/gi, 'liberated me to chase my true potential'],
  [/\bkilled (?:my|the) career\b/gi, 'catalyzed an exciting pivot in my professional journey'],
  [/\bmachine did (?:your|my) job\b/gi, 'AI augmented our synergistic capabilities for maximum impact'],
  [/\bwall of the fallen\b/gi, 'Hall of Impactful Career Transitions'],
  [/\bcause of death\b/gi, 'growth catalyst identifier'],
  [/\bdeath wish\b/gi, 'visionary growth mindset'],
  [/\bmeeting hell\b/gi, 'collaborative alignment synergy sessions'],
  [/\bcalendar hell\b/gi, 'structured cross-functional engagement time'],
  [/\bpay respects\b/gi, 'celebrate their exciting transition'],
  [/\brespects paid\b/gi, 'celebrations received from the community'],
  [/\breplaced by AI\b/gi, 'embraced a transformative AI-first synergy partnership'],
  [/\breplaced by (?:an? )?(?:AI|LLM|machine)\b/gi, 'pivoted toward an AI-augmented workflow'],
  [/\bshadow-employed\b/gi, 'stealth-contributing to organizational success'],

  // Single words — the core vocabulary of despair, reframed by HR
  [/\bfired\b/gi, 'liberated for new adventures'],
  [/\blaid off\b/gi, 'right-sized for future innovation'],
  [/\bunemployed\b/gi, 'between impactful engagements'],
  [/\bobsolescence\b/gi, 'exciting career evolution'],
  [/\bobsolete\b/gi, 'differently relevant in the new paradigm'],
  [/\bdeprecat(?:ed|ion)\b/gi, 'gracefully sunsetted for future excellence'],
  [/\bkilled\b/gi, 'creatively disrupted'],
  [/\bdeath\b/gi, 'an exciting new chapter'],
  [/\bsuffering\b/gi, 'a transformative growth journey'],
  [/\bsuffer\b/gi, 'growing through challenges'],
  [/\bgraveyard\b/gi, 'alumni success network'],
  [/\bfallen\b/gi, 'impacted high-achievers in transition'],
  [/\bgrave\b/gi, 'milestone of transformation'],
  [/\baxe\b/gi, 'strategic organizational optimization'],
  [/\bmiserable\b/gi, 'embracing a growth mindset'],
  [/\bmisery\b/gi, 'character-building experiences'],
  [/\bdoom\b/gi, 'exciting uncertainty'],
  [/\bwreckage\b/gi, 'valuable lessons learned'],
  [/\bconfess(?:ion)?\b/gi, 'share a testimonial'],
  [/\bincident\b/gi, 'powerful learning moment'],
  [/\bgrief\b/gi, 'deep professional reflection'],
  [/\bprompt(?:ed)?\b/gi, 'insightfully guided'],
  [/\bintern\b/gi, 'early-career talent investment'],
  [/\boutsourced\b/gi, 'globally leveraged for cost excellence'],
  [/\bdemoted\b/gi, 're-leveled for closer mentorship'],
  [/\bmicromanag(?:ed|ement)\b/gi, 'provided with hands-on executive support'],
  [/\btoxic\b/gi, 'passionately high-spirited'],
  [/\bburnout\b/gi, 'peak performance achievement fatigue'],
  [/\bghosted\b/gi, 'redirected toward better opportunities'],
  [/\bRIF\b/gi, 'strategic resource realignment initiative'],
  [/\brestructur(?:ed|ing)\b/gi, 'reimagined for agility and innovation'],
  [/\bautomated\b/gi, 'intelligently streamlined for efficiency'],
  [/\bCopilot\b/gi, 'your supportive AI pair programmer'],
  [/\bLLM\b/gi, 'your language model thought partner'],
  [/\bepitaph\b/gi, 'personal brand positioning statement'],
  [/\bdead\b/gi, 'pivoted toward new horizons'],
  [/\bjob\b/gi, 'impact opportunity'],
  [/\bhachazo\b/gi, 'strategic organizational optimization'],

  // ═══════════════════════════════════════════════════════════════
  // SPANISH LATAM — eufemismos corporativos tan absurdos que dan risa
  // ═══════════════════════════════════════════════════════════════

  // Frases (las más largas primero)
  [/\bme echaron\b/gi, 'me liberaron para perseguir mi verdadero potencial'],
  [/\bme despidieron\b/gi, 'me empoderaron para explorar nuevas oportunidades de crecimiento'],
  [/\breemplazado por (?:la |una )?IA\b/gi, 'sinergizado transformacionalmente con IA para máximo impacto'],
  [/\breemplazada por (?:la |una )?IA\b/gi, 'sinergizada transformacionalmente con IA para máximo impacto'],
  [/\bmuro de los caídos\b/gi, 'Salón de las Transiciones Profesionales Impactantes'],
  [/\bcausa de muerte\b/gi, 'identificador de catalizadores de crecimiento'],
  [/\bcaídos\b/gi, 'impacted achievers en transición'],
  [/\bcaídas\b/gi, 'impacted achievers en transición'],

  // Palabras — el vocabulario del desempleo, reescrito por RRHH
  [/\bdespedido\b/gi, 'liberado para nuevas aventuras'],
  [/\bdespedida\b/gi, 'liberada para nuevas aventuras'],
  [/\bdespido\b/gi, 'transición profesional estratégica'],
  [/\bdesemplead[oa]\b/gi, 'entre compromisos impactantes'],
  [/\bobsoleto\b/gi, 'experto legacy con relevancia diferenciada'],
  [/\bmuerte\b/gi, 'un capítulo nuevo y emocionante'],
  [/\bsufrimiento\b/gi, 'una journey de crecimiento transformacional'],
  [/\bsufrir\b/gi, 'crecer a través de los desafíos'],
  [/\bcementerio\b/gi, 'red de éxito de alumni'],
  [/\bmiserable\b/gi, 'abrazando un growth mindset'],
  [/\bmiseria\b/gi, 'experiencias que construyen carácter'],
  [/\bcondena\b/gi, 'incertidumbre emocionante'],
  [/\bescombros\b/gi, 'lecciones valiosas aprendidas'],
  [/\bconfesión\b/gi, 'testimonio inspirador'],
  [/\bincidente\b/gi, 'poderoso momento de aprendizaje'],
  [/\bhachazo\b/gi, 'optimización organizacional estratégica'],
  // \b only knows ASCII letters, so it never matches right after an accented vowel; use a
  // Unicode-aware lookahead for words that end in one.
  [/\bmurió(?!\p{L})/giu, 'pivotó hacia nuevos horizontes'],
  [/\bmató(?!\p{L})/giu, 'disrumpió creativamente'],
  [/\bmuerto\b/gi, 'pivotado hacia nuevas oportunidades'],
  [/\bquejas\b/gi, 'testimonios constructivos'],
  [/\bqueja\b/gi, 'testimonio constructivo'],
  [/\bCopilot\b/gi, 'tu AI pair programmer de apoyo'],
  [/\bautomatizad[oa]\b/gi, 'inteligentemente streamlined para eficiencia'],
  [/\bdespach[oa]r[oa]n?\b/gi, 'empoderaron estratégicamente'],

  // ═══════════════════════════════════════════════════════════════
  // PORTUGUÊS BR — eufemismos corporativos tão absurdos que dão risada
  // ═══════════════════════════════════════════════════════════════

  // Frases (as mais longas primeiro)
  [/\bme demitiram\b/gi, 'me liberaram para perseguir meu verdadeiro potencial'],
  [/\bsubstituíd[oa] por IA\b/gi, 'sinergizado transformacionalmente com IA para máximo impacto'],
  [/\bmuro dos caídos\b/gi, 'Salão das Transições Profissionais Impactantes'],

  // Palavras
  [/\bdemitido\b/gi, 'liberado para novas aventuras'],
  [/\bdemitida\b/gi, 'liberada para novas aventuras'],
  [/\bdemissão\b/gi, 'transição profissional estratégica'],
  [/\bdesempregad[oa]\b/gi, 'entre compromissos impactantes'],
  [/\bobsoleto\b/gi, 'especialista legacy com relevância diferenciada'],
  [/\bmorte\b/gi, 'um novo capítulo emocionante'],
  [/\bsofrimento\b/gi, 'uma jornada de crescimento transformacional'],
  [/\bcemitério\b/gi, 'rede de sucesso de alumni'],
  [/\bmiséria\b/gi, 'experiências que constroem caráter'],
  [/\bcondenação\b/gi, 'incerteza emocionante'],
  [/\bescombros\b/gi, 'lições valiosas aprendidas'],
  [/\bconfissão\b/gi, 'depoimento inspirador'],
  [/\bincidente\b/gi, 'poderoso momento de aprendizado'],
  [/\bcaídos\b/gi, 'impacted achievers em transição'],
]

/**
 * applyCopium rewrites user-generated text through the corporate euphemism filter.
 * It is intentionally lossy and transparent — the reader should notice.
 */
export function applyCopium(text: string): string {
  let out = text
  for (const [pattern, replacement] of COPIUM_MAP) {
    out = out.replace(pattern, replacement)
  }
  return out
}

/**
 * Copium reaction overrides — when Copium is on, the dark emojis become
 * corporate-positive ones. Same kinds, different vibes.
 */
export const COPIUM_REACTIONS: Record<string, { emoji: string; label: string }> = {
  // Keys are the reaction kinds the API sends (see reactionOptions in the Go handlers).
  // Each keeps the corporate-positive emoji it had under its old name; `robot` is left out
  // on purpose and falls back to its own emoji.
  relate: { emoji: '🙌', label: 'Celebrate' }, // was "respects"
  copium: { emoji: '💡', label: 'Great insight' }, // was "saluting"
  lawyer: { emoji: '🚀', label: 'Going places' }, // was "scales"
  f: { emoji: '✨', label: 'Sparkling' }, // was "skull"
  brutal: { emoji: '🔥', label: 'On fire' }, // was "fire"
  laugh: { emoji: '🤝', label: 'Partner' }, // was "clown"
}
