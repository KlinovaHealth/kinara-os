import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = {
  title: "Surveillance des maladies et alerte précoce · Kinara OS",
  description: "Détection interdomaine des épidémies. Kinara OS détecte les signaux épidémiques des semaines avant la surveillance classique en corrélant les données de santé, logistique, agriculture et maritime.",
};

const SIGNAL_CHAIN = [
  { domain: "Maritime", color: "#0EA5E9", day: "Jour 0",  signal: "Un navire d'une région touchée dédouane au port. Le manifeste d'équipage est enregistré. Aucun signal d'alerte. Aucun domaine seul ne dispose du contexte.", detail: "Le domaine maritime enregistre chaque arrivée de navire, liste d'équipage et port d'origine. Isolément, c'est une donnée de routine. Sa signification n'émerge qu'en combinaison avec ce qui suit." },
  { domain: "Agriculture", color: "#F59E0B", day: "Jour 14", signal: "Un choc de sécurité alimentaire est enregistré dans trois districts intérieurs. Le domaine logistique constate une demande entrante inhabituelle.", detail: "La malnutrition et l'immunodépression précèdent les pics de maladies de deux à quatre semaines dans les schémas d'épidémies documentés. Un signal agricole de cette ampleur, depuis ces districts, est un précurseur connu. Aucun système de santé ne le voit à temps sans une requête interdomaine." },
  { domain: "Logistique", color: "#3B82F6", day: "Jour 18", signal: "Schéma de mouvement inhabituel signalé sur les routes intérieures: trafic supérieur à la référence depuis les districts concernés.", detail: "Les données de mouvement du domaine logistique montrent le déplacement de population avant que les établissements de santé ne l'enregistrent. Les agents de terrain voyageant entre districts, les commerçants informels changeant de routes. Tout cela apparaît d'abord dans les enregistrements de transport." },
  { domain: "Santé", color: "#10B981", day: "Jour 21", signal: "Un cluster de consultations augmente fortement dans les trois mêmes districts. Plusieurs établissements. Symptômes similaires à la présentation.", detail: "Les cliniques individuelles voient leur propre file d'attente. Sans signal inter-établissements, chacune traite un pic local comme un problème local. Le schéma est invisible dans les dossiers d'un seul établissement de santé." },
  { domain: "Kinara Core", color: "#E0561C", day: "Jour 21", signal: "Schéma interdomaine soulevé. Quatre signaux corrélés entre maritime, agriculture, logistique et santé. Alerte précoce émise au dépositaire santé.", detail: "Kinara Core corrèle les quatre signaux sous gouvernance. Chaque jointure est autorisée, le minimum de données est divulgué, et l'alerte est attribuée dans le journal d'audit. Le dépositaire santé reçoit l'avertissement avec la chaîne de preuves. Aucune donnée brute d'un autre domaine n'est divulguée." },
];

const WHY_IT_MATTERS = [
  { heading: "La surveillance traditionnelle est monodomaine.", body: "Les systèmes d'information sanitaire suivent les visites en clinique et les résultats de laboratoire. Ils ne peuvent pas voir le choc de sécurité alimentaire qui a précédé le cluster de deux semaines. Les signaux existent. La connexion n'existe pas." },
  { heading: "La collecte de données est déjà faite.", body: "Kinara OS ne nécessite pas de nouveau programme de surveillance. Les enregistrements maritimes, agricoles, logistiques et sanitaires qui produisent l'alerte précoce transitent déjà par le système chaque jour. Le signal d'épidémie est une requête, pas une nouvelle source de données." },
  { heading: "Chaque jointure est gouvernée et attribuée.", body: "Corréler des données de santé avec des données logistiques ou agricoles est une requête interdomaine. Kinara Core vérifie la base légale, exécute avec un minimum de divulgation et consigne l'entrée d'audit avant que le résultat soit retourné." },
  { heading: "Des semaines, pas des jours.", body: "La chaîne de signaux maritime-à-santé dans les schémas d'épidémies documentés couvre 14 à 28 jours. Un système qui voit le signal maritime au Jour 0 et le corrèle avec les données agricoles du Jour 14 émet un avertissement au Jour 21. La surveillance monodomaine émet le même avertissement aux Jours 28 à 35." },
];

const USE_CASES = [
  { label: "Préparation aux épidémies", color: "#10B981", desc: "Corrélez les données de clusters en clinique avec les mouvements logistiques et les arrivées maritimes pour détecter les épidémies 7 à 14 jours plus tôt que la surveillance basée uniquement sur les cliniques." },
  { label: "Sécurité alimentaire et corrélation maladies", color: "#F59E0B", desc: "Associez les chocs de rendement agricole aux résultats de santé en aval. Identifiez les populations à risque avant que la malnutrition ne se présente cliniquement." },
  { label: "Cartographie des vecteurs et de la transmission", color: "#3B82F6", desc: "Utilisez les données de mouvement logistique pour modéliser les voies de transmission des maladies. Identifiez les corridors de transport qui corrèlent avec les schémas de propagation." },
  { label: "Contrôle sanitaire portuaire", color: "#0EA5E9", desc: "Croisez les manifestes d'équipage maritime et les données de port d'origine avec les zones d'épidémies actives pour éclairer les décisions sanitaires portuaires." },
  { label: "Préparation de la chaîne d'approvisionnement", color: "#E0561C", desc: "Quand une alerte sanitaire est émise, le domaine logistique détient déjà les niveaux de stock en entrepôt et la disponibilité des véhicules. Le dispatch de réponse commence avant le dépôt de la demande formelle." },
];

export default function FrEarlyWarningPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="relative overflow-hidden px-8 pt-24 pb-0" style={{ background: "radial-gradient(ellipse 70% 60% at 80% 0%, rgba(16,185,129,0.12) 0%, transparent 55%), radial-gradient(ellipse 50% 60% at 10% 90%, rgba(14,165,233,0.06) 0%, transparent 55%), #080F1E" }}>
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="inline-flex items-center gap-2 mb-8">
              <span className="w-6 h-px bg-[#10B981]" />
              <p className="text-[12px] font-[600] uppercase tracking-[0.18em] text-[#8A98A8]">Surveillance des maladies · Alerte précoce</p>
            </div>
            <h1 className="text-[64px] md:text-[96px] lg:text-[112px] font-[800] tracking-[-0.05em] leading-[0.91] mb-8 max-w-5xl">
              <span className="text-white block">Une épidémie détectée</span>
              <span className="block" style={{ background: "linear-gradient(135deg, #10B981 0%, #34D399 100%)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent", backgroundClip: "text" }}>
                avant que quiconque ne la nomme.
              </span>
            </h1>
            <div className="grid md:grid-cols-2 gap-10 mb-16 max-w-5xl">
              <p className="text-[18px] md:text-[20px] text-[#8A98A8] leading-relaxed">
                Kinara OS détecte les signaux épidémiques des semaines avant la surveillance monodomaine en corrélant les données de santé, logistique, agriculture et maritime sous gouvernance. Aucune nouvelle collecte de données. Aucune base de données partagée. Chaque jointure autorisée et consignée.
              </p>
              <div className="flex flex-col gap-4 items-start self-end pb-1">
                <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors">
                  Demander un entretien
                </Link>
                <Link href="/fr/domains/health" className="text-[14px] font-[600] text-[#8A98A8] hover:text-white transition-colors">
                  Voir le domaine Santé &rarr;
                </Link>
              </div>
            </div>
          </div>
          <div className="h-20 bg-gradient-to-b from-transparent to-white pointer-events-none" />
        </section>

        {/* ── Pourquoi ──────────────────────────────────────────── */}
        <section className="bg-white px-8 py-24 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Le problème</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Les signaux existent. La connexion n'existe pas.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-6">
                  Chaque épidémie documentée possède un signal précurseur qui apparaît dans des données autres que le système de santé: dans les prix alimentaires, les schémas de transport, les arrivées portuaires. Ces données existent. Elles sont collectées quotidiennement. Elles résident dans des systèmes séparés avec des dépositaires séparés.
                </p>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  La surveillance traditionnelle attend que le système de santé voie suffisamment de cas pour déclencher une alerte. Entre-temps, l'épidémie est déjà dans la communauté. Kinara OS voit le signal corrélé sur les quatre domaines avant que le système de santé ait un cluster à signaler.
                </p>
              </div>
              <div className="space-y-0 border border-[#C3CEDA]">
                {WHY_IT_MATTERS.map((item, i) => (
                  <div key={i} className="bg-white px-7 py-6 border-b border-[#C3CEDA] last:border-0">
                    <p className="text-[16px] font-[700] text-[#0F1B2D] mb-2">{item.heading}</p>
                    <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Chaîne de signaux ─────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-24 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-14">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">Comment ça marche</p>
              <h2 className="text-[40px] md:text-[56px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] max-w-3xl">
                Suivez un signal d'épidémie à travers quatre domaines.
              </h2>
            </div>
            <div className="space-y-0 border border-[#C3CEDA]">
              {SIGNAL_CHAIN.map((row, i) => (
                <div
                  key={i}
                  className="grid md:grid-cols-2 border-b border-[#C3CEDA] last:border-0"
                  style={{ borderLeftWidth: "3px", borderLeftStyle: "solid", borderLeftColor: row.color, background: i === 4 ? "rgba(224,86,28,0.03)" : "white" }}
                >
                  <div className="px-8 py-6 md:border-r border-[#C3CEDA]">
                    <p className="text-[11px] font-[700] uppercase tracking-[0.15em] mb-1" style={{ color: row.color }}>{row.domain}</p>
                    <p className="text-[12px] text-[#8A98A8] mb-3">{row.day} depuis le premier signal</p>
                    <p className={`text-[16px] font-[700] leading-snug ${i === 4 ? "text-[#E0561C]" : "text-[#0F1B2D]"}`}>{row.signal}</p>
                  </div>
                  <div className="px-8 py-6">
                    <p className="text-[14px] text-[#5D6E82] leading-relaxed">{row.detail}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Applications ──────────────────────────────────────── */}
        <section className="bg-white px-8 py-24 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-14">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">Applications</p>
              <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] max-w-2xl">Ce que cela permet entre domaines.</h2>
            </div>
            <div className="grid md:grid-cols-3 gap-px bg-[#C3CEDA]">
              {USE_CASES.map((uc) => (
                <div key={uc.label} className="bg-white p-8">
                  <div className="w-8 h-1 mb-5 rounded-full" style={{ background: uc.color }} />
                  <p className="text-[17px] font-[800] text-[#0F1B2D] mb-3 tracking-[-0.02em]">{uc.label}</p>
                  <p className="text-[14px] text-[#5D6E82] leading-relaxed">{uc.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Gouvernance ───────────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-24 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Gouvernance des données</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Aucun domaine ne cède la garde pour produire l'alerte.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-6">
                  Le dépositaire maritime ne partage pas ses manifestes d'équipage avec le système de santé. Le dépositaire agricole ne partage pas ses enregistrements de rendement avec le système logistique. Chaque domaine conserve la pleine garde de ses données.
                </p>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  Kinara Core pose une question gouvernée à travers la frontière, reçoit une réponse à divulgation minimale et émet l'alerte. Chaque étape est dans le journal d'audit.
                </p>
              </div>
              <div style={{ background: "#080F1E" }} className="relative p-10 overflow-hidden">
                <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
                <div className="relative space-y-5">
                  <p className="text-[11px] font-[700] uppercase tracking-[0.14em] text-[#8A98A8]">Entrée d'audit · Requête d'alerte précoce</p>
                  {[
                    { label: "Requête émise par", value: "Kinara Core · Moteur de patterns" },
                    { label: "Domaines interrogés", value: "Maritime, Agriculture, Logistique, Santé" },
                    { label: "Données divulguées", value: "Minimum: signaux agrégés uniquement" },
                    { label: "Données brutes divulguées", value: "Aucune" },
                    { label: "Base légale vérifiée", value: "Oui · Réf. politique 04-EW" },
                    { label: "Alerte émise à", value: "Dépositaire santé uniquement" },
                    { label: "Entrée d'audit", value: "Immuable · Ne peut être modifiée" },
                  ].map((row, i) => (
                    <div key={i} className="flex items-start justify-between gap-8 border-b border-[#1A2A40] pb-4 last:border-0 last:pb-0">
                      <p className="text-[13px] text-[#5D6E82]">{row.label}</p>
                      <p className="text-[13px] font-[600] text-white text-right">{row.value}</p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ── CTA ──────────────────────────────────────────────── */}
        <section className="relative px-8 py-24 overflow-hidden" style={{ background: "radial-gradient(ellipse 60% 80% at 30% 50%, rgba(16,185,129,0.12) 0%, transparent 60%), #080F1E" }}>
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#10B981] mb-6">Demander une démonstration</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  Nous exécuterons la requête interdomaine en direct.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">Un entretien de 45 minutes. Nous parcourons la chaîne de signaux, montrons la couche de gouvernance autorisant la jointure et affichons le journal d'audit en temps réel. Rien de mis en scène.</p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors">
                  Demander un entretien
                </Link>
                <Link href="/fr/governance" className="inline-block border border-[#3A4F68] text-[#8A98A8] text-[15px] font-[600] px-9 py-4 hover:border-[#8A98A8] hover:text-white transition-colors">
                  Découvrir le modèle de gouvernance
                </Link>
              </div>
            </div>
          </div>
        </section>

      </main>
      <Footer lang="fr" />
    </>
  );
}
