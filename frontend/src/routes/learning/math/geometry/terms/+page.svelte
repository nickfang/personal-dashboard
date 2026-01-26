<script>
  export let data;
  const { terms } = data;
</script>

<main>
  <h1>Geometry Terms</h1>
  <p>
    Geometry is the branch of mathematics that deals with the properties, measurement, and
    relationships of points, lines, angles, surfaces, and solids. The terms in this section are the
    basic building blocks of geometry.
  </p>
  <p>Click on a term to learn more about it.</p>
  <ul>
    {#each Object.entries(terms) as [term, description]}
      <li>
        <strong>{term}</strong> - {description}
      </li>
    {/each}
  </ul>
</main>
