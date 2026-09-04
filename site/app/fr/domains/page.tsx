import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = { title: "Domaines · Kinara OS" };

const DOMAINS = [
  { name: "Santé",       color: "#059669", services: 40, steward: "Ministère de la Santé",       desc: "Patients, visites, laboratoires, vaccination, détection des épidémies.", href: "/fr/domains/health" },
  { name: "Agriculture", color: "#D97706", services: 40, steward: "Ministère de l'Agriculture",  desc: "Parcelles, intrants, récolte, prix du marché, subventions.", href: "/fr/domains/agriculture" },
  { name: "Logistique",  color: "#2563EB", services: 36, steward: "Ministère des Transports",    desc: "Itinéraires, livraisons, dommages, retours, prévisions.", href: "/fr/domains/logistics" },
  { name: "Maritime",    color: "#0891B2", services: 36, steward: "Autorité portuaire",           desc: "Navires, postes à quai, cargaisons, douanes, financement du commerce.", href: "/fr/domains/maritime" },
];

export default function FrDomainsPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Domaines</p>
          <h1 className="text-[64px] md:text-[80px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-4xl">
            Quatre domaines. 152 services. Une couche de coordination.
          </h1>
          <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl mb-16">
            Chaque domaine est souverain. Chaque dépositaire conserve la pleine garde de ses enregistrements. Kinara OS régit ce qui franchit les frontières.
          </p>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
            {DOMAINS.map((d) => (
              <Link key={d.name} href={d.href} className="group block border border-[#C3CEDA] p-6 bg-white hover:border-[#E0561C] transition-colors" style={{ borderTop: `4px solid ${d.color}` }}>
                <h3 className="text-[22px] font-[800] tracking-[-0.02em] text-[#0F1B2D] mb-1">{d.name}</h3>
                <p className="text-[12px] font-[600] uppercase tracking-[0.1em] text-[#8A98A8] mb-3">{d.services} services · {d.steward}</p>
                <p className="text-[14px] text-[#5D6E82] leading-relaxed mb-4">{d.desc}</p>
                <span className="text-[13px] font-[600] text-[#E0561C] group-hover:underline">Voir le domaine &rarr;</span>
              </Link>
            ))}
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-10 max-w-2xl">Les jointures importantes franchissent les frontières de domaine.</h2>
            <div className="grid md:grid-cols-2 gap-4">
              {[
                { from: "Santé", to: "Logistique", example: "Une rupture de stock dans une clinique déclenche une requête logistique. Qu'y a-t-il dans l'entrepôt le plus proche ? Un véhicule est-il disponible ?" },
                { from: "Agriculture", to: "Logistique", example: "Un enregistrement de récolte déclenche une demande de transport. Quelle quantité de produits doit être déplacée, depuis où, et pour quand ?" },
                { from: "Santé", to: "Agriculture", example: "Un indicateur nutritionnel dans les données de santé correspond à un signal de défaillance des récoltes dans les données agricoles." },
                { from: "Maritime", to: "Logistique", example: "Un conteneur est dédouané au port. Une instruction logistique est générée sans transfert manuel." },
              ].map((item) => (
                <div key={item.from + item.to} className="bg-white border border-[#C3CEDA] rounded-lg p-6">
                  <p className="text-[12px] font-[800] uppercase tracking-[0.1em] text-[#E0561C] mb-3">{item.from} &rarr; {item.to}</p>
                  <p className="text-[15px] text-[#5D6E82] leading-relaxed">{item.example}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="max-w-[1440px] mx-auto px-8 py-20">
          <div className="grid md:grid-cols-2 gap-16 items-end">
            <h2 className="text-[52px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">Voir les domaines lors d&apos;un entretien.</h2>
            <div>
              <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-8">Nous vous montrerons l&apos;architecture des domaines et une requête interdomaine en direct.</p>
              <a href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-8 py-4 hover:bg-[#c94d19] transition-colors">Demander un entretien</a>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
