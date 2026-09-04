import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "À propos · Kinara OS" };

export default function FrAboutPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">À propos</p>
          <h1 className="text-[64px] md:text-[80px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-4xl">
            Conçu pour résoudre le problème de coordination. Pas pour remplacer les ministères qui l&apos;ont.
          </h1>
          <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
            Kinara OS est conçu et détenu par Klinova. Nous travaillons avec des institutions publiques africaines pour bâtir une infrastructure de coordination qui respecte et renforce la souveraineté institutionnelle.
          </p>
        </section>

        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-2xl">Nous ne voulons pas vos données. Nous voulons gouverner la jointure.</h2>
            <div className="text-[18px] text-[#5D6E82] leading-relaxed space-y-4 max-w-2xl">
              <p>L&apos;approche standard de la coordination des données du secteur public demande aux institutions de centraliser leurs données dans une plateforme commune. C&apos;est pourquoi cette approche échoue toujours.</p>
              <p>Notre position est différente. Chaque institution conserve ses propres systèmes et ses propres données. Ce que nous opérons, c&apos;est la couche qui régit quand et comment une question peut franchir les frontières entre elles.</p>
              <p>Ce n&apos;est pas un compromis. C&apos;est le seul modèle qui fonctionne dans un environnement multiministériel et multijuridictionnel où la souveraineté sur les enregistrements est non négociable.</p>
            </div>
          </div>
        </section>

        <section className="max-w-[1440px] mx-auto px-8 py-20">
          <div className="grid md:grid-cols-2 gap-16 items-end">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">Parlez-nous.</h2>
            <div>
              <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-8">Apportez-nous le problème de coordination. Nous vous dirons directement si et comment Kinara OS y répond.</p>
              <a href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-8 py-4 hover:bg-[#c94d19] transition-colors">Demander un entretien</a>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
