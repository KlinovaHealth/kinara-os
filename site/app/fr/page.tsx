"use client";

import Image from "next/image";
import Link from "next/link";
import { useEffect } from "react";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

function useReveal() {
  useEffect(() => {
    const els = document.querySelectorAll(".reveal");
    const io = new IntersectionObserver(
      (entries) => entries.forEach((e) => { if (e.isIntersecting) { e.target.classList.add("visible"); io.unobserve(e.target); } }),
      { threshold: 0.06 }
    );
    els.forEach((el) => io.observe(el));
    return () => io.disconnect();
  }, []);
}

const STORIES = [
  { label: "Santé", color: "#10B981", headline: "Une rupture évitée avant que personne ne s'en aperçoive.", body: "Une infirmière dans une clinique rurale saisit les niveaux de stock sur un téléphone basique. Kinara OS lit la pénurie, soulève une requête logistique, la vérifie contre l'entrepôt le plus proche et dépêche un véhicule. Aucun appel téléphonique, aucun tableur, aucun ministère ne cédant ses dossiers." },
  { label: "Agriculture", color: "#F59E0B", headline: "Une récolte qui se déplace d'elle-même.", body: "Un rendement confirmé devient une demande de transport au moment où il est enregistré. Quelle quantité, depuis quel district, vers quel marché. Le domaine logistique reçoit le signal sous gouvernance. Le grain se déplace avant que la fenêtre se ferme." },
  { label: "Logistique", color: "#3B82F6", headline: "Un véhicule acheminé avant que quiconque ne le demande.", body: "Un entrepôt enregistre un surplus. Une clinique enregistre une pénurie. Kinara OS soulève la correspondance, vérifie la politique et achemine le véhicule. Aucun coordinateur au milieu. Aucun appel téléphonique. L'enregistrement de la décision est dans le journal avant que le chauffeur parte." },
  { label: "Maritime", color: "#0EA5E9", headline: "Marchandise libérée. Camion dépêché. Aucun appel passé.", body: "Un conteneur dédouané au port. Une instruction logistique transmise vers l'intérieur avant que le conteneur quitte le quai. Aucun transfert manuel. Aucune expédition manquante. Deux opérateurs coordonnés sans qu'aucun ne voie les données de l'autre." },
];

const DOMAINS = [
  { name: "Santé",       services: 40, desc: "Patients, visites, laboratoires, vaccination, détection d'épidémies.", color: "#10B981", href: "/fr/domains/health" },
  { name: "Agriculture", services: 40, desc: "Parcelles, intrants, récolte, prix du marché, subvention.", color: "#F59E0B", href: "/fr/domains/agriculture" },
  { name: "Logistique",  services: 36, desc: "Acheminement, livraison, entreposage, retours, prévisions.", color: "#3B82F6", href: "/fr/domains/logistics" },
  { name: "Maritime",    services: 36, desc: "Navires, postes à quai, cargaisons, douanes, financement du commerce.", color: "#0EA5E9", href: "/fr/domains/maritime" },
];

const STEPS = [
  { n: "01", head: "Une infirmière compte le rayon.", body: "Stock saisi sur un téléphone basique via WhatsApp ou SMS. Aucune application, aucun réseau requis. Attribué à un agent de santé nommé." },
  { n: "02", head: "L'enregistrement franchit la périphérie.", body: "Conservé localement, mis en file d'attente, transmis et réconcilié au retour du signal. Aucun enregistrement n'est jamais perdu." },
  { n: "03", head: "La jointure est autorisée.", body: "La rupture de stock en santé soulève une question logistique. Kinara Core vérifie la base légale, autorise la jointure et consigne l'entrée d'audit." },
  { n: "04", head: "La logistique répond.", body: "L'entrepôt le détient. Un véhicule est acheminé. Déplacement autorisé. Déplacement consigné." },
  { n: "05", head: "L'infirmière est informée.", body: "La réponse arrive sur le même téléphone: une référence d'expédition et une fenêtre d'arrivée. Temps écoulé: quelques minutes." },
];

export default function FrHome() {
  useReveal();

  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="relative overflow-hidden" style={{ background: "radial-gradient(ellipse 80% 60% at 75% -10%, rgba(224,86,28,0.22) 0%, transparent 60%), radial-gradient(ellipse 60% 60% at 10% 90%, rgba(14,165,233,0.08) 0%, transparent 55%), #080F1E" }}>
          <div className="absolute inset-0 dot-grid pointer-events-none" />
          <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[#E0561C] to-transparent" />
          <div className="relative max-w-[1440px] mx-auto px-8 pt-24 pb-0">
            <div className="inline-flex items-center gap-2 mb-8">
              <span className="w-6 h-px bg-[#E0561C]" />
              <p className="text-[12px] font-[600] uppercase tracking-[0.18em] text-[#8A98A8]">L'infrastructure numérique de l'Afrique</p>
            </div>
            <div className="mb-8 max-w-5xl">
              <h1 className="text-[72px] md:text-[108px] lg:text-[130px] font-[800] tracking-[-0.05em] leading-[0.91]">
                <span className="text-white block">L'Afrique fonctionne</span>
                <span className="text-white block">avec des données</span>
                <span className="block" style={{ background: "linear-gradient(135deg, #FF6B35 0%, #E0561C 40%, #FF9A6C 100%)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent", backgroundClip: "text" }}>
                  qui ne se connectent jamais.
                </span>
              </h1>
            </div>
            <div className="grid md:grid-cols-2 gap-10 mb-16 max-w-5xl">
              <p className="text-[18px] md:text-[20px] text-[#8A98A8] leading-relaxed">
                Kinara OS est une couche de coordination à gouvernance politique et juridiction-aware qui permet aux services de santé, d'agriculture, de logistique et de maritime de fonctionner de manière indépendante tout en ne partageant que des informations autorisées, auditables et limitées à leur finalité.
              </p>
              <div className="flex flex-col sm:flex-row md:flex-col lg:flex-row gap-4 items-start self-end pb-1">
                <Link href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors whitespace-nowrap">
                  Demander un entretien
                </Link>
                <Link href="/fr/platform" className="link-arrow inline-flex items-center text-[14px] font-[600] text-[#8A98A8] hover:text-white transition-colors whitespace-nowrap py-3.5">
                  Voir l'architecture <span>&rarr;</span>
                </Link>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-y-2 border-t border-[#1A2A40] pt-6 mb-0 pb-10">
              {["152 services en production", "4 domaines gouvernés", "144 bases de données isolées, une par service", "2 locataires actifs"].map((item, i) => (
                <span key={i} className="text-[13px] text-[#5D6E82] flex items-center">
                  {i > 0 && <span className="mx-4 text-[#1A2A40]">&middot;</span>}
                  {item}
                </span>
              ))}
            </div>
          </div>

          {/* Architecture diagram */}
          <div className="relative max-w-[1440px] mx-auto px-8">
            <div className="relative overflow-hidden" style={{ border: "1px solid rgba(255,255,255,0.08)", background: "#0D1828", boxShadow: "0 0 80px rgba(224,86,28,0.08), 0 40px 80px rgba(0,0,0,0.4)" }}>
              <div className="absolute left-0 top-0 bottom-0 w-14 border-r border-[#1A2A40] flex items-end pb-6 z-10">
                <p className="text-[9px] font-[600] uppercase tracking-[0.18em] text-[#3A4F68] whitespace-nowrap" style={{ writingMode: "vertical-rl", transform: "rotate(180deg)" }}>
                  Kinara OS &nbsp;&rarr;&nbsp; Propulsé par Kinara Core
                </p>
              </div>
              <div className="pl-14">
                <Image src="/diagrams/architecture-fr.png" alt="Système Kinara OS: quatre domaines souverains coordonnés par Kinara Core" width={1600} height={760} className="w-full h-auto" priority />
              </div>
            </div>
            <div className="grid grid-cols-2 border-x border-b border-[#1A2A40]">
              <Link href="/fr/contact" className="flex items-center justify-between px-6 py-4 border-r border-[#1A2A40] text-[13px] font-[600] text-[#5D6E82] hover:text-white hover:bg-[#0D1828] transition-colors">
                Demander un entretien <span className="text-[#E0561C]">&rarr;</span>
              </Link>
              <Link href="/fr/platform" className="flex items-center justify-between px-6 py-4 text-[13px] font-[600] text-[#5D6E82] hover:text-white hover:bg-[#0D1828] transition-colors">
                Voir l'architecture <span className="text-[#E0561C]">&rarr;</span>
              </Link>
            </div>
          </div>
          <div className="h-24 bg-gradient-to-b from-transparent to-white pointer-events-none" />
        </section>

        {/* ── Quatre histoires ──────────────────────────────────── */}
        <section className="bg-white px-8 py-24">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-16 reveal">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">Pourquoi ça compte</p>
              <h2 className="text-[48px] md:text-[64px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] max-w-2xl">
                Quatre histoires. Un système.
              </h2>
            </div>
            <div className="grid sm:grid-cols-2 md:grid-cols-4 gap-px bg-[#C3CEDA]">
              {STORIES.map((s, i) => (
                <div key={i} className="bg-white p-8 group card-lift cursor-default">
                  <div className="inline-flex items-center gap-2 mb-6 px-3 py-1 text-[11px] font-[700] uppercase tracking-[0.14em]" style={{ background: s.color + "15", color: s.color }}>
                    {s.label}
                  </div>
                  <h3 className="text-[22px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-snug mb-4">{s.headline}</h3>
                  <p className="text-[15px] text-[#5D6E82] leading-relaxed">{s.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Alerte précoce ────────────────────────────────────── */}
        <section className="relative px-8 py-24 overflow-hidden" style={{ background: "#080F1E" }}>
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto reveal">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Alerte précoce</p>
                <h2 className="text-[40px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-8">
                  Une épidémie détectée avant que quiconque ne la nomme.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-6">
                  Un cluster de consultations en clinique augmente dans trois districts. Un enregistrement logistique montre un mouvement inhabituel depuis la même zone. Un enregistrement agricole révèle un choc de sécurité alimentaire deux semaines auparavant. Un navire d'une région touchée a dédouané au port dix jours plus tôt.
                </p>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-8">
                  Aucun domaine seul ne voit le schéma. Kinara OS le voit. Le signal existe dans les données qui transitent déjà par le système chaque jour. Il ne nécessite pas un nouvel effort de collecte de données. Il nécessite une façon gouvernée de poser la question à travers les frontières.
                </p>
                <p className="text-[15px] font-[700] text-white border-l-2 border-[#E0561C] pl-5">
                  C'est des semaines plus tôt que la surveillance traditionnelle ne le détecte.
                </p>
                <div className="mt-8">
                  <Link href="/fr/early-warning" className="inline-flex items-center gap-2 text-[14px] font-[700] text-[#10B981] hover:text-[#34D399] transition-colors">
                    Voir la chaîne de signaux complète &rarr;
                  </Link>
                </div>
              </div>
              <div className="space-y-2">
                <div className="flex items-center gap-4 px-5 pb-1">
                  <div className="flex-shrink-0 w-24 text-right">
                    <span className="text-[10px] font-[600] uppercase tracking-[0.15em] text-[#3A4F68]">Jours écoulés</span>
                  </div>
                  <div className="flex-shrink-0 w-px" />
                  <div className="flex-1">
                    <span className="text-[10px] font-[600] uppercase tracking-[0.15em] text-[#3A4F68]">Signal</span>
                  </div>
                </div>
                {[
                  { domain: "Maritime", color: "#0EA5E9", signal: "Navire d'une région touchée dédouane au port", day: "Jour 0" },
                  { domain: "Agriculture", color: "#F59E0B", signal: "Choc de sécurité alimentaire enregistré dans trois districts", day: "Jour 14" },
                  { domain: "Logistique", color: "#3B82F6", signal: "Schéma de mouvement inhabituel signalé sur les routes intérieures", day: "Jour 18" },
                  { domain: "Santé", color: "#10B981", signal: "Cluster de consultations en clinique en hausse dans les mêmes districts", day: "Jour 21" },
                  { domain: "Kinara Core", color: "#E0561C", signal: "Schéma interdomaine soulevé. Alerte précoce émise.", day: "Jour 21" },
                ].map((row, i) => (
                  <div key={i} className="flex items-start gap-4 p-5 border border-[#1A2A40]" style={{ background: i === 4 ? "rgba(224,86,28,0.08)" : "rgba(26,42,64,0.4)", borderColor: i === 4 ? "rgba(224,86,28,0.3)" : undefined }}>
                    <div className="flex-shrink-0 w-24 text-right">
                      <span className="text-[11px] font-[700] text-[#3A4F68]">{row.day}</span>
                    </div>
                    <div className="flex-shrink-0 w-px self-stretch" style={{ background: row.color }} />
                    <div className="flex-1">
                      <p className="text-[10px] font-[700] uppercase tracking-[0.14em] mb-1" style={{ color: row.color }}>{row.domain}</p>
                      <p className={`text-[14px] leading-snug ${i === 4 ? "text-white font-[700]" : "text-[#5D6E82]"}`}>{row.signal}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Couche de gouvernance ─────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-24 border-y border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start reveal">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Le système</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  La couche de gouvernance Kinara
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-6">
                  Kinara Core se situe à l'intersection de quatre domaines souverains. Il ne détient jamais de données; il régit les questions qui les traversent. Chaque jointure est vérifiée contre la politique, exécutée avec un minimum de divulgation et consignée dans un journal d'audit immuable.
                </p>
                <Link href="/fr/governance" className="link-arrow text-[14px] font-[700] text-[#E0561C] hover:text-[#c94d19] transition-colors">
                  Découvrir le modèle de gouvernance <span>&rarr;</span>
                </Link>
              </div>
              <div className="space-y-2">
                {DOMAINS.map((d) => (
                  <Link key={d.name} href={d.href} className="group flex items-center gap-4 bg-white border border-[#C3CEDA] p-5 card-lift block">
                    <div className="flex-shrink-0 w-1 self-stretch rounded-full" style={{ background: d.color }} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-baseline gap-3 mb-0.5">
                        <p className="text-[17px] font-[800] text-[#0F1B2D] tracking-[-0.02em]">{d.name}</p>
                        <p className="text-[12px] font-[600] text-[#8A98A8] uppercase tracking-[0.08em]">{d.services} services</p>
                      </div>
                      <p className="text-[13px] text-[#5D6E82]">{d.desc}</p>
                    </div>
                    <span className="text-[20px] text-[#C3CEDA] group-hover:text-[#E0561C] transition-colors flex-shrink-0">&rarr;</span>
                  </Link>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Chiffres ──────────────────────────────────────────── */}
        <section className="bg-white px-8 py-24">
          <div className="max-w-[1440px] mx-auto reveal">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-px bg-[#C3CEDA] border border-[#C3CEDA]">
              {[
                { n: "152", l: "services", sub: "en production" },
                { n: "144", l: "bases de données isolées", sub: "une par service" },
                { n: "4",   l: "domaines", sub: "entièrement gouvernés" },
                { n: "72h", l: "d'enregistrements", sub: "conservés hors ligne à la périphérie" },
              ].map((s) => (
                <div key={s.n} className="bg-white p-10">
                  <p className="text-[64px] font-[800] tracking-[-0.05em] leading-none mb-1" style={{ color: "#E0561C" }}>{s.n}</p>
                  <p className="text-[18px] font-[700] text-[#0F1B2D] tracking-[-0.02em]">{s.l}</p>
                  <p className="text-[13px] text-[#8A98A8] mt-0.5">{s.sub}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Comment ça marche ─────────────────────────────────── */}
        <section className="relative px-8 py-24 overflow-hidden" style={{ background: "radial-gradient(ellipse 70% 80% at 90% 50%, rgba(224,86,28,0.10) 0%, transparent 60%), #080F1E" }}>
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-50" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start reveal">
              <div className="md:sticky md:top-32">
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Comment ça marche</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-8">
                  Une requête. Cinq étapes. Aucune garde cédée.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-8">
                  Suivez une demande de stock d'une infirmière dans une clinique rurale jusqu'à un ordre de dispatch logistique. Regardez le journal d'audit se remplir sans qu'aucun ministère ne voie les données brutes d'un autre.
                </p>
                <blockquote className="border-l-2 border-[#E0561C] pl-5">
                  <p className="text-[18px] font-[700] text-white leading-snug">
                    Temps écoulé: quelques minutes. Personne n'a déposé de demande. Aucun ministère n'a cédé la garde.
                  </p>
                </blockquote>
              </div>
              <div className="space-y-2">
                {STEPS.map((step, i) => (
                  <div key={i} className="flex gap-5 p-6 border border-[#1A2A40] hover:border-[#E0561C]/30 transition-colors" style={{ background: "rgba(26,42,64,0.4)" }}>
                    <div className="flex-shrink-0">
                      <span className="text-[11px] font-[800] block mt-0.5" style={{ color: "#E0561C" }}>{step.n}</span>
                    </div>
                    <div>
                      <p className="text-[16px] font-[700] text-white mb-1.5 tracking-[-0.01em]">{step.head}</p>
                      <p className="text-[14px] text-[#5D6E82] leading-relaxed">{step.body}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
          <div className="absolute bottom-0 left-0 right-0 h-20 bg-gradient-to-b from-transparent to-white pointer-events-none" />
        </section>

        {/* ── Deux locataires ───────────────────────────────────── */}
        <section className="bg-white px-8 py-24">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center reveal">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">En production</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Deux locataires. Un système actif.
                </h2>
                <div className="mb-8 pl-5 border-l-2 border-[#E0561C]">
                  <p className="text-[12px] font-[700] uppercase tracking-[0.12em] text-[#E0561C] mb-2">Klinova &nbsp;&middot;&nbsp; Opérateur commercial</p>
                  <p className="text-[16px] text-[#5D6E82] leading-relaxed">Klinova est le principal locataire commercial sur Kinara OS, connectant cliniques, hôpitaux, assureurs, pharmacies, médecins et réseaux de livraison dans un système de santé coordonné. Chaque fonctionnalité que Klinova déploie fonctionne sur la même infrastructure gouvernée disponible pour les futurs locataires.</p>
                </div>
                <div className="mb-10 pl-5 border-l-2 border-[#10B981]">
                  <p className="text-[12px] font-[700] uppercase tracking-[0.12em] text-[#10B981] mb-2">Village Health Access &nbsp;&middot;&nbsp; Opérateur non lucratif</p>
                  <p className="text-[16px] text-[#5D6E82] leading-relaxed">Un réseau de cliniques communautaires en activité. De vrais patients. De vrais dossiers. Stock saisi via WhatsApp par des agents de terrain identifiés. Rapports tirés directement de la base de données opérationnelle. Aucun export manuel, aucun tableur.</p>
                </div>
                <Link href="/fr/evidence" className="link-arrow text-[15px] font-[700] text-[#E0561C] hover:text-[#c94d19] transition-colors">
                  Voir les preuves <span>&rarr;</span>
                </Link>
              </div>
              {/* Audit log */}
              <div className="bg-[#F5F8FB] border border-[#C3CEDA] p-8">
                <p className="text-[11px] font-[700] uppercase tracking-[0.14em] text-[#8A98A8] mb-6">Journal d'audit · Village Health Access</p>
                <div className="space-y-3">
                  {[
                    { ts: "08:14:03", action: "STOCK_ENTRY", worker: "A. Diallo · Clinique 04", status: "OK" },
                    { ts: "08:14:09", action: "CROSS_DOMAIN_JOIN", worker: "Kinara Core → Logistique", status: "AUTORISÉ" },
                    { ts: "08:14:11", action: "DISPATCH_RAISED", worker: "Entrepôt E-07 → Véhicule V-12", status: "OK" },
                    { ts: "08:17:22", action: "DELIVERY_ETA_SENT", worker: "→ A. Diallo · Clinique 04", status: "OK" },
                  ].map((row, i) => (
                    <div key={i} className="flex items-center gap-4 py-3 border-b border-[#C3CEDA] last:border-0 font-mono text-[12px]">
                      <span className="text-[#8A98A8] flex-shrink-0 w-16">{row.ts}</span>
                      <span className="font-[700] text-[#0F1B2D] flex-shrink-0 w-40 truncate">{row.action}</span>
                      <span className="text-[#5D6E82] flex-1 truncate hidden sm:block">{row.worker}</span>
                      <span className="flex-shrink-0 text-[10px] font-[700] px-2 py-0.5" style={{ background: row.status === "AUTORISÉ" ? "#E0561C15" : "#10B98115", color: row.status === "AUTORISÉ" ? "#E0561C" : "#10B981" }}>{row.status}</span>
                    </div>
                  ))}
                </div>
                <p className="text-[12px] text-[#8A98A8] mt-5">Chaque entrée est immuable. Chaque jointure est attribuée.</p>
              </div>
            </div>
          </div>
        </section>

        {/* ── CTA ──────────────────────────────────────────────── */}
        <section className="relative px-8 py-28 overflow-hidden" style={{ background: "radial-gradient(ellipse 60% 80% at 30% 50%, rgba(224,86,28,0.18) 0%, transparent 60%), #080F1E" }}>
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto text-center reveal">
            <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-8">Connecter l'Afrique</p>
            <h2 className="text-[52px] md:text-[80px] lg:text-[96px] font-[800] tracking-[-0.05em] text-white leading-[0.93] mb-8 max-w-4xl mx-auto">
              Apportez-nous le problème qui franchit deux organisations.
            </h2>
            <p className="text-[18px] text-[#5D6E82] mb-12 max-w-xl mx-auto leading-relaxed">
              C'est précisément celui pour lequel nous sommes conçus. Un entretien dure 45 minutes. Nous vous montrerons le système en fonctionnement, pas des diapositives.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link href="/fr/contact" className="bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors">
                Demander un entretien
              </Link>
              <Link href="/fr/about" className="link-arrow text-[15px] font-[600] text-[#5D6E82] hover:text-white transition-colors py-4">
                À propos de Kinara OS <span>&rarr;</span>
              </Link>
            </div>
            <p className="text-[13px] text-[#3A4F68] mt-10">Kinara OS est conçu et détenu par Klinova &nbsp;&middot;&nbsp; kinaraos.com</p>
          </div>
        </section>

      </main>
      <Footer lang="fr" />
    </>
  );
}
